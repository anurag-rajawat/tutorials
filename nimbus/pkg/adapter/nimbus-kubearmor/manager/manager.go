package manager

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	kubearmorv1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorController/api/security.kubearmor.com/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/anurag-rajawat/tutorials/nimbus/adapter/nimbus-kubearmor/watcher"
	intentv1alpha1 "github.com/anurag-rajawat/tutorials/nimbus/api/v1alpha1"
	"github.com/anurag-rajawat/tutorials/nimbus/pkg/adapter/k8s"
)

type manager struct {
	logger    logr.Logger
	k8sClient client.Client
	scheme    *runtime.Scheme
	wg        *sync.WaitGroup
}

func (m *manager) run(ctx context.Context) {
	m.logger.Info("Starting manager")
	nimbusPolicyChan := make(chan *intentv1alpha1.NimbusPolicy)
	kspChan := make(chan *kubearmorv1.KubeArmorPolicy)

	watchr := watcher.NewWatcher(ctx)

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		watchr.RunNpWatcher(ctx, nimbusPolicyChan)
	}()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		watchr.RunKspWatcher(ctx, kspChan)
	}()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.managePolicies(ctx, nimbusPolicyChan, kspChan)
	}()

	m.logger.Info("Started manager")
	m.wg.Wait()
}

func Run(ctx context.Context) {
	logger := log.FromContext(ctx)
	scheme := runtime.NewScheme()

	utilruntime.Must(intentv1alpha1.AddToScheme(scheme))
	utilruntime.Must(kubearmorv1.AddToScheme(scheme))

	k8sClient, err := k8s.NewClient(scheme)
	if err != nil {
		logger.Error(err, "failed to initialize Kubernetes client")
		return
	}

	mgr := &manager{
		logger:    logger,
		k8sClient: k8sClient,
		wg:        &sync.WaitGroup{},
		scheme:    scheme,
	}

	mgr.run(ctx)
}
