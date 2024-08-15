package watcher

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"

	intentv1alpha1 "github.com/anurag-rajawat/tutorials/nimbus/api/v1alpha1"
)

func (w *Watcher) getNpInformer() cache.SharedIndexInformer {
	nimbusPolicyGvr := schema.GroupVersionResource{
		Group:    "intent.security.nimbus.com",
		Version:  "v1alpha1",
		Resource: "nimbuspolicies",
	}
	return w.factory.ForResource(nimbusPolicyGvr).Informer()
}

func (w *Watcher) watchNimbusPolicies(ctx context.Context, nimbusPolicyChan chan *intentv1alpha1.NimbusPolicy) {
	informer := w.getNpInformer()
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			u := obj.(*unstructured.Unstructured)
			if w.isOrphan(ctx, u.GetOwnerReferences(), "SecurityIntentBinding", u.GetNamespace()) {
				w.logger.V(4).Info("Ignoring orphan Policy", "name", u.GetName(), "namespace", u.GetNamespace(), "operation", "add")
				return
			}

			w.logger.V(4).Info("NimbusPolicy found", "nimbusPolicy.Name", u.GetName(), "nimbusPolicy.Namespace", u.GetNamespace())

			nimbusPolicy := &intentv1alpha1.NimbusPolicy{}
			w.decodePolicy(u, nimbusPolicy)
			nimbusPolicyChan <- nimbusPolicy
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldU := oldObj.(*unstructured.Unstructured)
			newU := newObj.(*unstructured.Unstructured)
			if w.isOrphan(ctx, newU.GetOwnerReferences(), "SecurityIntentBinding", newU.GetNamespace()) {
				w.logger.V(4).Info("Ignoring orphan Policy", "name", oldU.GetName(), "namespace", oldU.GetNamespace(), "operation", "update")
				return
			}

			if oldU.GetGeneration() != newU.GetGeneration() {
				w.logger.V(4).Info("NimbusPolicy updated", "nimbusPolicy.Name", newU.GetName(), "nimbusPolicy.Namespace", newU.GetNamespace())
				nimbusPolicy := &intentv1alpha1.NimbusPolicy{}
				w.decodePolicy(newU, nimbusPolicy)
				nimbusPolicyChan <- nimbusPolicy
			}
		},
	}

	_, err := informer.AddEventHandler(handlers)
	if err != nil {
		w.logger.Error(err, "failed to add nimbus policy handler")
		return
	}

	w.logger.Info("Started NimbusPolicy watcher")
	informer.Run(ctx.Done())
	close(nimbusPolicyChan)
	w.logger.Info("Stopped NimbusPolicy watcher")
}

func (w *Watcher) RunNpWatcher(ctx context.Context, nimbusPolicyChan chan *intentv1alpha1.NimbusPolicy) {
	w.logger.Info("Starting NimbusPolicy watcher")
	w.watchNimbusPolicies(ctx, nimbusPolicyChan)
}
