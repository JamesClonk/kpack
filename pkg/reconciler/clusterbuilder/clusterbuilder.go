package clusterBuilder

import (
	"context"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/controller"

	"github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	corev1alpha1 "github.com/pivotal/kpack/pkg/apis/core/v1alpha1"
	"github.com/pivotal/kpack/pkg/client/clientset/versioned"
	v1alpha1informers "github.com/pivotal/kpack/pkg/client/informers/externalversions/build/v1alpha1"
	v1alpha1Listers "github.com/pivotal/kpack/pkg/client/listers/build/v1alpha1"
	"github.com/pivotal/kpack/pkg/cnb"
	"github.com/pivotal/kpack/pkg/reconciler"
	"github.com/pivotal/kpack/pkg/registry"
	"github.com/pivotal/kpack/pkg/tracker"
)

const (
	ReconcilerName = "CustomBuilders"
	Kind           = "CustomBuilder"
)

type NewBuildpackRepository func(clusterStore *v1alpha1.ClusterStore) cnb.BuildpackRepository

type BuilderCreator interface {
	CreateBuilder(keychain authn.Keychain, buildpackRepo cnb.BuildpackRepository, clusterStack *v1alpha1.ClusterStack, spec v1alpha1.BuilderSpec) (v1alpha1.BuilderRecord, error)
}

func NewController(
	opt reconciler.Options,
	informer v1alpha1informers.ClusterBuilderInformer,
	repoFactory NewBuildpackRepository,
	builderCreator BuilderCreator,
	keychainFactory registry.KeychainFactory,
	clusterStoreInformer v1alpha1informers.ClusterStoreInformer,
	clusterStackInformer v1alpha1informers.ClusterStackInformer,
) *controller.Impl {
	c := &Reconciler{
		Client:               opt.Client,
		ClusterBuilderLister: informer.Lister(),
		RepoFactory:          repoFactory,
		BuilderCreator:       builderCreator,
		KeychainFactory:      keychainFactory,
		ClusterStoreLister:   clusterStoreInformer.Lister(),
		ClusterStackLister:   clusterStackInformer.Lister(),
	}
	impl := controller.NewImpl(c, opt.Logger, ReconcilerName)
	informer.Informer().AddEventHandler(reconciler.Handler(impl.Enqueue))

	c.Tracker = tracker.New(impl.EnqueueKey, opt.TrackerResyncPeriod())
	clusterStoreInformer.Informer().AddEventHandler(reconciler.Handler(c.Tracker.OnChanged))
	clusterStackInformer.Informer().AddEventHandler(reconciler.Handler(c.Tracker.OnChanged))

	return impl
}

type Reconciler struct {
	Client               versioned.Interface
	ClusterBuilderLister v1alpha1Listers.ClusterBuilderLister
	RepoFactory          NewBuildpackRepository
	BuilderCreator       BuilderCreator
	KeychainFactory      registry.KeychainFactory
	Tracker              reconciler.Tracker
	ClusterStoreLister   v1alpha1Listers.ClusterStoreLister
	ClusterStackLister   v1alpha1Listers.ClusterStackLister
}

func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	_, builderName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	builder, err := c.ClusterBuilderLister.Get(builderName)
	if k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	builder = builder.DeepCopy()

	builderRecord, creationError := c.reconcileBuilder(builder)
	if creationError != nil {
		builder.Status.ErrorCreate(creationError)

		err := c.updateStatus(builder)
		if err != nil {
			return err
		}

		return controller.NewPermanentError(creationError)
	}

	builder.Status.BuilderRecord(builderRecord)
	return c.updateStatus(builder)
}

func (c *Reconciler) reconcileBuilder(builder *v1alpha1.ClusterBuilder) (v1alpha1.BuilderRecord, error) {
	clusterStore, err := c.ClusterStoreLister.Get(builder.Spec.Store.Name)
	if err != nil {
		return v1alpha1.BuilderRecord{}, err
	}

	err = c.Tracker.Track(clusterStore, builder.NamespacedName())
	if err != nil {
		return v1alpha1.BuilderRecord{}, err
	}

	clusterStack, err := c.ClusterStackLister.Get(builder.Spec.Stack.Name)
	if err != nil {
		return v1alpha1.BuilderRecord{}, err
	}

	err = c.Tracker.Track(clusterStack, builder.NamespacedName())
	if err != nil {
		return v1alpha1.BuilderRecord{}, err
	}

	if !clusterStack.Status.GetCondition(corev1alpha1.ConditionReady).IsTrue() {
		return v1alpha1.BuilderRecord{}, errors.Errorf("stack %s is not ready", clusterStack.Name)
	}

	keychain, err := c.KeychainFactory.KeychainForSecretRef(registry.SecretRef{
		ServiceAccount: builder.Spec.ServiceAccountRef.Name,
		Namespace:      builder.Spec.ServiceAccountRef.Namespace,
	})
	if err != nil {
		return v1alpha1.BuilderRecord{}, err
	}

	return c.BuilderCreator.CreateBuilder(keychain, c.RepoFactory(clusterStore), clusterStack, builder.Spec.BuilderSpec)
}

func (c *Reconciler) updateStatus(desired *v1alpha1.ClusterBuilder) error {
	desired.Status.ObservedGeneration = desired.Generation

	original, err := c.ClusterBuilderLister.Get(desired.Name)
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(desired.Status, original.Status) {
		return nil
	}

	_, err = c.Client.KpackV1alpha1().ClusterBuilders().UpdateStatus(desired)
	return err
}
