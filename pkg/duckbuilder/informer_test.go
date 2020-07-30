package duckbuilder

import (
	"sync"
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	"github.com/pivotal/kpack/pkg/client/clientset/versioned/fake"
	"github.com/pivotal/kpack/pkg/client/informers/externalversions"
)

func TestDuckBuilderInformer(t *testing.T) {
	spec.Run(t, "TestDuckBuilderInformer", testDuckBuilderInformer)
}

func testDuckBuilderInformer(t *testing.T, when spec.G, it spec.S) {
	const (
		builderNamespace = "some-other-namespace"
		builderName      = "some-custom-builder"

		clusterBuilderName = "some-custom-cluster-builder"
	)

	var (
		stopCh = make(chan struct{})

		builder = &v1alpha1.Builder{
			ObjectMeta: metav1.ObjectMeta{
				Name:      builderName,
				Namespace: builderNamespace,
			},
			Spec: v1alpha1.NamespacedBuilderSpec{},
			Status: v1alpha1.BuilderStatus{
				BuilderMetadata: v1alpha1.BuildpackMetadataList{
					{
						Id:      "another-buildpack",
						Version: "another-version",
					},
				},
				Stack:       v1alpha1.BuildStack{},
				LatestImage: "",
			},
		}

		clusterBuilder = &v1alpha1.ClusterBuilder{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterBuilderName,
			},
			Spec: v1alpha1.ClusterBuilderSpec{},
			Status: v1alpha1.BuilderStatus{
				BuilderMetadata: v1alpha1.BuildpackMetadataList{
					{
						Id:      "another-buildpack",
						Version: "another-version",
					},
				},
				Stack:       v1alpha1.BuildStack{},
				LatestImage: "",
			},
		}
	)

	client := fake.NewSimpleClientset(
		builder,
		clusterBuilder,
	)

	factory := externalversions.NewSharedInformerFactory(client, 10*time.Hour)

	subject := DuckBuilderInformer{
		BuilderInformer:        factory.Kpack().V1alpha1().Builders(),
		ClusterBuilderInformer: factory.Kpack().V1alpha1().ClusterBuilders(),
	}
	duckBuilderLister := subject.Lister()
	factory.Start(stopCh)

	factory.WaitForCacheSync(stopCh)

	it.After(func() {
		close(stopCh)
	})

	when("#Lister", func() {
		it("can return a builder of type CustomBuilder", func() {
			duckBuilder, err := duckBuilderLister.Namespace(builderNamespace).Get(v1.ObjectReference{
				Kind:      v1alpha1.BuilderKind,
				Namespace: builderNamespace,
				Name:      builderName,
			})
			require.NoError(t, err)

			require.Equal(t, builder.ObjectMeta, duckBuilder.ObjectMeta)
			require.Equal(t, builder.Status, duckBuilder.Status)
			require.Equal(t, []v1.LocalObjectReference(nil), duckBuilder.Spec.ImagePullSecrets)
		})

		it("can return a builder of type CustomClusterBuilder", func() {
			duckBuilder, err := duckBuilderLister.Namespace("").Get(v1.ObjectReference{
				Kind: v1alpha1.ClusterBuilderKind,
				Name: clusterBuilderName,
			})
			require.NoError(t, err)

			require.Equal(t, clusterBuilder.ObjectMeta, duckBuilder.ObjectMeta)
			require.Equal(t, clusterBuilder.Status, duckBuilder.Status)
			require.Equal(t, []v1.LocalObjectReference(nil), duckBuilder.Spec.ImagePullSecrets)
		})

		it("returns a k8s not found error on missing builder", func() {
			for _, typ := range []string{
				v1alpha1.ClusterBuilderKind,
				v1alpha1.BuilderKind,
			} {
				_, err := duckBuilderLister.Namespace("some-namespace").Get(v1.ObjectReference{
					Kind: typ,
					Name: "doesnt-exisit",
				})
				require.True(t, k8serrors.IsNotFound(err))
			}
		})

		it("returns an error for unknown Kind", func() {
			_, err := duckBuilderLister.Namespace("some-namespace").Get(v1.ObjectReference{
				Kind: "unknown",
				Name: "doesnt-exisit",
			})
			require.EqualError(t, err, "unknown builder type: unknown")
		})
	})

	when("#AddEventHandler", func() {
		it("adds the event handler to each builder's informer", func() {
			testHandler := &testHandler{}
			subject.AddEventHandler(testHandler)

			assert.Eventually(t, func() bool {
				return len(testHandler.added) == 2
			}, 5*time.Second, time.Millisecond)

			assert.Contains(t, testHandler.added, builder)
			assert.Contains(t, testHandler.added, clusterBuilder)
		})
	})
}

type testHandler struct {
	added []interface{}
	sync.Mutex
}

func (t *testHandler) OnAdd(obj interface{}) {
	t.Lock()
	defer t.Unlock()
	t.added = append(t.added, obj)
}

func (t *testHandler) OnUpdate(oldObj, newObj interface{}) {
}

func (t *testHandler) OnDelete(obj interface{}) {
}
