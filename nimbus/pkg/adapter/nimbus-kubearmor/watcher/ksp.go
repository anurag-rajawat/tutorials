package watcher

import (
	"context"

	kubearmorv1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorController/api/security.kubearmor.com/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

func (w *Watcher) getKspInformer() cache.SharedIndexInformer {
	kspGvr := schema.GroupVersionResource{
		Group:    "security.kubearmor.com",
		Version:  "v1",
		Resource: "kubearmorpolicies",
	}
	return w.factory.ForResource(kspGvr).Informer()
}

func (w *Watcher) watchKsp(ctx context.Context, kspChan chan *kubearmorv1.KubeArmorPolicy) {
	informer := w.getKspInformer()
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldU := oldObj.(*unstructured.Unstructured)
			newU := newObj.(*unstructured.Unstructured)

			if w.isOrphan(ctx, newU.GetOwnerReferences(), "NimbusPolicy", newU.GetNamespace()) {
				w.logger.V(4).Info("Ignoring orphan Policy", "name", oldU.GetName(), "namespace", oldU.GetNamespace(), "operation", "update")
				return
			}

			if oldU.GetGeneration() != newU.GetGeneration() {
				w.logger.V(4).Info("KubeArmorPolicy modified", "kubeArmorPolicy.Name", newU.GetName(), "kubeArmorPolicy.Namespace", newU.GetNamespace())
				ksp := &kubearmorv1.KubeArmorPolicy{}
				w.decodePolicy(newU, ksp)
				kspChan <- ksp
			}
		},
		DeleteFunc: func(obj interface{}) {
			u := obj.(*unstructured.Unstructured)
			if w.isOrphan(ctx, u.GetOwnerReferences(), "NimbusPolicy", u.GetNamespace()) {
				w.logger.V(4).Info("Ignoring orphan Policy", "name", u.GetName(), "namespace", u.GetNamespace(), "operation", "delete")
				return
			}

			w.logger.V(4).Info("KubeArmorPolicy deleted", "kubeArmorPolicy.Name", u.GetName(), "kubeArmorPolicy.Namespace", u.GetNamespace())

			ksp := &kubearmorv1.KubeArmorPolicy{}
			w.decodePolicy(u, ksp)
			kspChan <- ksp
		},
	})
	if err != nil {
		w.logger.Error(err, "failed to add KubeArmorPolicy event handler")
		return
	}

	w.logger.Info("Started KubeArmorPolicy watcher")
	informer.Run(ctx.Done())

	close(kspChan)
	w.logger.Info("Stopped KubeArmorPolicy watcher")
}

func (w *Watcher) RunKspWatcher(ctx context.Context, kspChan chan *kubearmorv1.KubeArmorPolicy) {
	w.logger.Info("Starting KubeArmorPolicy watcher")
	w.watchKsp(ctx, kspChan)
}
