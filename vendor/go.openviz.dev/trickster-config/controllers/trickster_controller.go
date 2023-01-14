/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
	"os"
	"path/filepath"

	"github.com/trickstercache/trickster/v2/cmd/trickster/config"
	bo "github.com/trickstercache/trickster/v2/pkg/backends/options"
	rule "github.com/trickstercache/trickster/v2/pkg/backends/rule/options"
	cache "github.com/trickstercache/trickster/v2/pkg/cache/options"
	tracing "github.com/trickstercache/trickster/v2/pkg/observability/tracing/options"
	rwopts "github.com/trickstercache/trickster/v2/pkg/proxy/request/rewriter/options"
	trickstercachev1alpha1 "go.openviz.dev/trickster-config/api/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// TODO(tamal): Must be configurable
const configDir = "/tmp/trickster"

// TricksterReconciler reconciles a Trickster object
type TricksterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Fn     func(cfg *config.Config) error
}

//+kubebuilder:rbac:groups=trickstercache.org,resources=tricksters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=trickstercache.org,resources=tricksters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=trickstercache.org,resources=tricksters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Trickster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *TricksterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var trickster trickstercachev1alpha1.Trickster
	if err := r.Get(ctx, req.NamespacedName, &trickster); err != nil {
		log.Error(err, "unable to fetch Trickster")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cfg := config.NewConfig()
	//delete(cfg.Caches, "default")
	//delete(cfg.Backends, "default")
	//delete(cfg.NegativeCacheConfigs, "default")
	//delete(cfg.TracingConfigs, "default")
	if trickster.Spec.Main != nil {
		cfg.Main = trickster.Spec.Main
	}
	if trickster.Spec.Nats != nil {
		cfg.Nats = trickster.Spec.Nats
	}
	if trickster.Spec.Secret != nil {
		err := r.writeConfig(ctx, req.Name, trickster.Spec.Secret)
		if err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}
	if trickster.Spec.Frontend != nil {
		cfg.Frontend = trickster.Spec.Frontend
	}
	if trickster.Spec.Logging != nil {
		cfg.Logging = trickster.Spec.Logging
	}
	if trickster.Spec.Metrics != nil {
		cfg.Metrics = trickster.Spec.Metrics
	}
	if trickster.Spec.NegativeCacheConfigs != nil {
		cfg.NegativeCacheConfigs = trickster.Spec.NegativeCacheConfigs
	}
	if trickster.Spec.ReloadConfig != nil {
		cfg.ReloadConfig = trickster.Spec.ReloadConfig
	}
	{
		var list trickstercachev1alpha1.BackendList
		sel := labels.Everything()
		if trickster.Spec.BackendSelector != nil {
			var err error
			sel, err = metav1.LabelSelectorAsSelector(trickster.Spec.BackendSelector)
			if err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		}
		if err := r.List(context.Background(), &list, client.InNamespace(req.Namespace), client.MatchingLabelsSelector{Selector: sel}); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		if cfg.Backends == nil {
			cfg.Backends = make(map[string]*bo.Options, len(list.Items))
		}
		for _, item := range list.Items {
			if item.Spec.Secret != nil {
				err := r.writeConfig(ctx, req.Name, item.Spec.Secret)
				if err != nil {
					return ctrl.Result{}, client.IgnoreNotFound(err)
				}
			}
			cfg.Backends[item.Name] = &item.Spec.Options
		}
	}
	{
		var list trickstercachev1alpha1.CacheList
		sel := labels.Everything()
		if trickster.Spec.CacheSelector != nil {
			var err error
			sel, err = metav1.LabelSelectorAsSelector(trickster.Spec.CacheSelector)
			if err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		}
		if err := r.List(context.Background(), &list, client.InNamespace(req.Namespace), client.MatchingLabelsSelector{Selector: sel}); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		if cfg.Caches == nil {
			cfg.Caches = make(map[string]*cache.Options, len(list.Items))
		}
		for _, item := range list.Items {
			if item.Spec.Secret != nil {
				err := r.writeConfig(ctx, req.Name, item.Spec.Secret)
				if err != nil {
					return ctrl.Result{}, client.IgnoreNotFound(err)
				}
			}
			cfg.Caches[item.Name] = &item.Spec.Options
		}
	}
	{
		var list trickstercachev1alpha1.RequestRewriterList
		sel := labels.Everything()
		if trickster.Spec.RequestRewriterSelector != nil {
			var err error
			sel, err = metav1.LabelSelectorAsSelector(trickster.Spec.RequestRewriterSelector)
			if err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		}
		if err := r.List(context.Background(), &list, client.InNamespace(req.Namespace), client.MatchingLabelsSelector{Selector: sel}); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		if cfg.RequestRewriters == nil {
			cfg.RequestRewriters = make(map[string]*rwopts.Options, len(list.Items))
		}
		for _, item := range list.Items {
			cfg.RequestRewriters[item.Name] = &item.Spec.Options
		}
	}
	{
		var list trickstercachev1alpha1.RuleList
		sel := labels.Everything()
		if trickster.Spec.RuleSelector != nil {
			var err error
			sel, err = metav1.LabelSelectorAsSelector(trickster.Spec.RuleSelector)
			if err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		}
		if err := r.List(context.Background(), &list, client.InNamespace(req.Namespace), client.MatchingLabelsSelector{Selector: sel}); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		if cfg.Rules == nil {
			cfg.Rules = make(map[string]*rule.Options, len(list.Items))
		}
		for _, item := range list.Items {
			cfg.Rules[item.Name] = &item.Spec.Options
		}
	}
	{
		var list trickstercachev1alpha1.TracingConfigList
		sel := labels.Everything()
		if trickster.Spec.TracingConfigSelector != nil {
			var err error
			sel, err = metav1.LabelSelectorAsSelector(trickster.Spec.TracingConfigSelector)
			if err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		}
		if err := r.List(context.Background(), &list, client.InNamespace(req.Namespace), client.MatchingLabelsSelector{Selector: sel}); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		if cfg.TracingConfigs == nil {
			cfg.TracingConfigs = make(map[string]*tracing.Options, len(list.Items))
		}
		for _, item := range list.Items {
			if item.Spec.Secret != nil {
				err := r.writeConfig(ctx, req.Name, item.Spec.Secret)
				if err != nil {
					return ctrl.Result{}, client.IgnoreNotFound(err)
				}
			}
			cfg.TracingConfigs[item.Name] = &item.Spec.Options
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	fmt.Println(string(data))

	if err := r.Fn(cfg); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

func (r *TricksterReconciler) writeConfig(ctx context.Context, ns string, sp *core.SecretProjection) error {
	var secret core.Secret
	err := r.Get(ctx, client.ObjectKey{Namespace: ns, Name: sp.Name}, &secret)
	if err != nil {
		return err
	}
	for _, item := range sp.Items {
		path := filepath.Join(configDir, item.Path)
		err = os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			return err
		}
		err = os.WriteFile(filepath.Join(configDir, item.Path), secret.Data[item.Key], 0444)
		if err != nil {
			return err
		}
	}
	return nil
}

var (
	tricksterSecretKey = ".trickster.secret"
)

// SetupWithManager sets up the controller with the Manager.
func (r *TricksterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// https://book.kubebuilder.io/cronjob-tutorial/controller-implementation.html#setup
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &trickstercachev1alpha1.Trickster{}, tricksterSecretKey, func(rawObj client.Object) []string {
		trickster := rawObj.(*trickstercachev1alpha1.Trickster)
		secretNames := sets.NewString()

		if trickster.Spec.Secret != nil {
			secretNames.Insert(trickster.Spec.Secret.Name)
		}
		{
			var list trickstercachev1alpha1.BackendList
			sel := labels.Everything()
			if trickster.Spec.BackendSelector != nil {
				var err error
				sel, err = metav1.LabelSelectorAsSelector(trickster.Spec.BackendSelector)
				if err != nil {
					return secretNames.UnsortedList()
				}
			}
			if err := r.List(context.Background(), &list, client.InNamespace(trickster.Namespace), client.MatchingLabelsSelector{Selector: sel}); err != nil {
				return secretNames.UnsortedList()
			}
			for _, item := range list.Items {
				if item.Spec.Secret != nil {
					secretNames.Insert(item.Spec.Secret.Name)
				}
			}
		}
		{
			var list trickstercachev1alpha1.CacheList
			sel := labels.Everything()
			if trickster.Spec.CacheSelector != nil {
				var err error
				sel, err = metav1.LabelSelectorAsSelector(trickster.Spec.CacheSelector)
				if err != nil {
					return secretNames.UnsortedList()
				}
			}
			if err := r.List(context.Background(), &list, client.InNamespace(trickster.Namespace), client.MatchingLabelsSelector{Selector: sel}); err != nil {
				return secretNames.UnsortedList()
			}
			for _, item := range list.Items {
				if item.Spec.Secret != nil {
					secretNames.Insert(item.Spec.Secret.Name)
				}
			}
		}
		{
			var list trickstercachev1alpha1.TracingConfigList
			sel := labels.Everything()
			if trickster.Spec.TracingConfigSelector != nil {
				var err error
				sel, err = metav1.LabelSelectorAsSelector(trickster.Spec.TracingConfigSelector)
				if err != nil {
					return secretNames.UnsortedList()
				}
			}
			if err := r.List(context.Background(), &list, client.InNamespace(trickster.Namespace), client.MatchingLabelsSelector{Selector: sel}); err != nil {
				return secretNames.UnsortedList()
			}
			for _, item := range list.Items {
				if item.Spec.Secret != nil {
					secretNames.Insert(item.Spec.Secret.Name)
				}
			}
		}

		return secretNames.UnsortedList()
	}); err != nil {
		return err
	}
	secretHandler := func(a client.Object) []reconcile.Request {
		var tricksters trickstercachev1alpha1.TricksterList
		if err := r.List(context.Background(), &tricksters, client.InNamespace(a.GetNamespace()), client.MatchingFields{tricksterSecretKey: a.GetName()}); err != nil {
			return nil
		}
		var req []reconcile.Request
		for _, o := range tricksters.Items {
			req = append(req, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&o)})
		}
		return req
	}

	handlerGenerator := func(selectorGetter func(c *trickstercachev1alpha1.Trickster) *metav1.LabelSelector) handler.EventHandler {
		return handler.EnqueueRequestsFromMapFunc(func(a client.Object) []reconcile.Request {
			var tricksters trickstercachev1alpha1.TricksterList
			if err := r.List(context.Background(), &tricksters, client.InNamespace(a.GetNamespace())); err != nil {
				return nil
			}
			var req []reconcile.Request
			for _, c := range tricksters.Items {
				sel, err := metav1.LabelSelectorAsSelector(selectorGetter(&c))
				if err != nil {
					return nil
				}
				if sel.Matches(labels.Set(a.GetLabels())) {
					req = append(req, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&c)})
				}
			}
			return req
		})
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&trickstercachev1alpha1.Trickster{}).
		Watches(&source.Kind{Type: &trickstercachev1alpha1.Backend{}}, handlerGenerator(func(c *trickstercachev1alpha1.Trickster) *metav1.LabelSelector {
			return c.Spec.BackendSelector
		})).
		Watches(&source.Kind{Type: &trickstercachev1alpha1.Cache{}}, handlerGenerator(func(c *trickstercachev1alpha1.Trickster) *metav1.LabelSelector {
			return c.Spec.CacheSelector
		})).
		Watches(&source.Kind{Type: &trickstercachev1alpha1.RequestRewriter{}}, handlerGenerator(func(c *trickstercachev1alpha1.Trickster) *metav1.LabelSelector {
			return c.Spec.RequestRewriterSelector
		})).
		Watches(&source.Kind{Type: &trickstercachev1alpha1.Rule{}}, handlerGenerator(func(c *trickstercachev1alpha1.Trickster) *metav1.LabelSelector {
			return c.Spec.RuleSelector
		})).
		Watches(&source.Kind{Type: &trickstercachev1alpha1.TracingConfig{}}, handlerGenerator(func(c *trickstercachev1alpha1.Trickster) *metav1.LabelSelector {
			return c.Spec.TracingConfigSelector
		})).
		Watches(&source.Kind{Type: &core.Secret{}}, handler.EnqueueRequestsFromMapFunc(secretHandler)).
		Complete(r)
}
