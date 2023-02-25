package main

import (
	"fmt"
	"os"
	goruntime "runtime"
	"strings"
	"sync"

	"github.com/trickstercache/trickster/v2/cmd/trickster/config"
	"github.com/trickstercache/trickster/v2/cmd/trickster/config/validate"
	"github.com/trickstercache/trickster/v2/pkg/cache"
	tl "github.com/trickstercache/trickster/v2/pkg/observability/logging"
	"github.com/trickstercache/trickster/v2/pkg/observability/metrics"
	"github.com/trickstercache/trickster/v2/pkg/runtime"
	"github.com/trickstercache/trickster/v2/pkg/util/yamlx"

	trickstercachev1alpha1 "go.openviz.dev/trickster-config/api/v1alpha1"
	"go.openviz.dev/trickster-config/controllers"
	krt "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"
)

var (
	scheme   = krt.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	once sync.Once
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(trickstercachev1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func startKubeController(conf *config.Config, wg *sync.WaitGroup, log *tl.Logger,
	caches map[string]cache.Cache, args []string,
) {
	if conf == nil || conf.Resources == nil {
		return
	}
	// assumes all parameters are instantiated

	once.Do(func() {
		go func() {
			var metricsAddr string = ":8080"
			var enableLeaderElection bool
			var probeAddr string = ":8081"
			// flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
			// flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
			// flag.BoolVar(&enableLeaderElection, "leader-elect", false,
			//	"Enable leader election for controller manager. "+
			//		"Enabling this will ensure there is only one active controller manager.")
			// klog.InitFlags(flag.CommandLine)
			// flag.Parse()

			ctrl.SetLogger(klog.NewKlogr())

			mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
				Scheme:                 scheme,
				MetricsBindAddress:     metricsAddr,
				Port:                   9443,
				HealthProbeBindAddress: probeAddr,
				LeaderElection:         enableLeaderElection,
				LeaderElectionID:       "95c6021a.org",
				// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
				// when the Manager ends. This requires the binary to immediately end when the
				// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
				// speeds up voluntary leader transitions as the new leader don't have to wait
				// LeaseDuration time first.
				//
				// In the default scaffold provided, the program ends immediately after
				// the manager stops, so would be fine to enable this option. However,
				// if you are doing or is intended to do any operation such as perform cleanups
				// after the manager stops then its usage might be unsafe.
				// LeaderElectionReleaseOnCancel: true,
			})
			if err != nil {
				setupLog.Error(err, "unable to start manager")
				os.Exit(1)
			}

			if err = (&controllers.TricksterReconciler{
				Client: mgr.GetClient(),
				Scheme: mgr.GetScheme(),
				Fn: func(nc *config.Config) error {
					yml, err := yaml.Marshal(nc)
					if err != nil {
						return err
					}
					md, err := yamlx.GetKeyList(string(yml))
					if err != nil {
						nc.SetDefaults(yamlx.KeyLookup{})
						return err
					}
					err = nc.SetDefaults(md)
					if err != nil {
						return err
						// nc.Main.configFilePath = flags.ConfigPath
						// c.Main.configLastModified = c.CheckFileLastModified()
					}
					reloadFF(nc, conf, log, wg, caches, args)
					return nil
				},
			}).SetupWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to create controller", "controller", "Trickster")
				os.Exit(1)
			}
			setupLog.Info("starting manager")
			if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
				setupLog.Error(err, "problem running manager")
				os.Exit(1)
			}
		}()
	})
}

func reloadFF(conf, oldConf *config.Config, log *tl.Logger, wg *sync.WaitGroup, caches map[string]cache.Cache, args []string) {
	var err error
	oldConf.Main.ReloaderLock.Lock()
	// if oldConf.IsStale() {
	tl.Warn(log, "configuration reload starting now", tl.Pairs{"source": "sighup"})
	err = applyConfig(conf, oldConf, wg, log, caches, args, nil)
	if err == nil {
		oldConf.Main.ReloaderLock.Unlock()
		return // runConfig will start a new HupMonitor in place of this one
	}
	//}
	//oldConf.Main.ReloaderLock.Unlock()
	//p := tl.Pairs{}
	//if err != nil {
	//	p["err"] = err.Error()
	//}
	//tl.Warn(log, "configuration NOT reloaded", p)
	return
}

func runConfig2(oldConf *config.Config, wg *sync.WaitGroup, logger *tl.Logger,
	oldCaches map[string]cache.Cache, args []string, errorFunc func(),
) error {
	metrics.BuildInfo.WithLabelValues(goruntime.Version(),
		applicationGitCommitID, applicationVersion).Set(1)

	cfgLock.Lock()
	defer cfgLock.Unlock()
	var err error

	sargs := make([]string, 0, len(args))
	// this sanitizes the args from -test flags, which can cause issues with unit tests relying on cli args
	for _, v := range args {
		if !strings.HasPrefix(v, "-test.") {
			sargs = append(sargs, v)
		}
	}

	// load the config
	conf, flags, err := config.Load(runtime.ApplicationName, runtime.ApplicationVersion, sargs)
	if err != nil {
		fmt.Println("\nERROR: Could not load configuration:", err.Error())
		if flags != nil && !flags.ValidateConfig {
			PrintUsage()
		}
		handleStartupIssue("", nil, nil, errorFunc)
		return err
	}

	// if it's a -version command, print version and exit
	if flags.PrintVersion {
		PrintVersion()
		return nil
	}

	err = validate.ValidateConfig(conf)
	if err != nil {
		handleStartupIssue("ERROR: Could not load configuration: "+err.Error(),
			nil, nil, errorFunc)
	}
	if flags.ValidateConfig {
		fmt.Println("Trickster configuration validation succeeded.")
		return nil
	}

	return applyConfig(conf, oldConf, wg, logger, oldCaches, args, errorFunc)
}
