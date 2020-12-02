package v1alpha2

import (
	"encoding/json"
	"strconv"

	"github.com/google/go-containerregistry/pkg/name"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/kmeta"

	"github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	corev1alpha1 "github.com/pivotal/kpack/pkg/apis/core/v1alpha1"
)

func (*Build) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Build")
}

func (b *Build) Tag() string {
	return b.Spec.Tags[0]
}

func (b *Build) ServiceAccount() string {
	return b.Spec.ServiceAccount
}

func (b *Build) BuilderSpec() v1alpha1.BuildBuilderSpec {
	return b.Spec.Builder
}

func (b *Build) Services() Services {
	return b.Spec.Services
}

func (b *Build) V1Alpha1Bindings() (v1alpha1.Bindings, error) {
	if b.Annotations == nil || b.Annotations[V1Alpha1BindingsAnnotation] == "" {
		return v1alpha1.Bindings{}, nil
	}
	var v1Bindings v1alpha1.Bindings
	if err := json.Unmarshal([]byte(b.Annotations[V1Alpha1BindingsAnnotation]), &v1Bindings); err != nil {
		return v1alpha1.Bindings{}, err
	}
	return v1Bindings, nil
}

func (b *Build) IsRunning() bool {
	if b == nil {
		return false
	}

	return b.Status.GetCondition(corev1alpha1.ConditionSucceeded).IsUnknown()
}

func (b *Build) BuildRef() string {
	if b == nil {
		return ""
	}

	return b.GetName()
}

func (b *Build) BuildReason() string {
	if b == nil {
		return ""
	}

	return b.GetAnnotations()[BuildReasonAnnotation]
}

func (b *Build) ImageGeneration() int64 {
	if b == nil {
		return 0
	}
	generation, ok := b.Labels[ImageGenerationLabel]
	if !ok {
		return 0
	}
	atoi, err := strconv.Atoi(generation)
	if err != nil {
		return 0
	}

	return int64(atoi)
}

func (b *Build) Stack() string {
	if b == nil {
		return ""
	}
	if !b.IsSuccess() {
		return ""
	}
	return b.Status.Stack.ID
}

func (b *Build) BuiltImage() string {
	if b == nil {
		return ""
	}
	if !b.IsSuccess() {
		return ""
	}

	return b.Status.LatestImage
}

func (b *Build) IsSuccess() bool {
	if b == nil {
		return false
	}
	return b.Status.GetCondition(corev1alpha1.ConditionSucceeded).IsTrue()
}

func (b *Build) IsFailure() bool {
	if b == nil {
		return false
	}
	return b.Status.GetCondition(corev1alpha1.ConditionSucceeded).IsFalse()
}

func (b *Build) PodName() string {
	return kmeta.ChildName(b.Name, "-build-pod")
}

func (b *Build) MetadataReady(pod *corev1.Pod) bool {
	return !b.Status.GetCondition(corev1alpha1.ConditionSucceeded).IsTrue() &&
		pod.Status.Phase == "Succeeded"
}

func (b *Build) Finished() bool {
	return !b.Status.GetCondition(corev1alpha1.ConditionSucceeded).IsUnknown()
}

func (b *Build) rebasable(builderStack string) bool {
	return b.Spec.LastBuild != nil &&
		b.Annotations[BuildReasonAnnotation] == BuildReasonStack && b.Spec.LastBuild.StackId == builderStack
}

func (b *Build) builtWithStack(runImage string) bool {
	if b.Status.Stack.RunImage == "" {
		return false
	}

	lastBuildRunImageRef, err := name.ParseReference(b.Status.Stack.RunImage)
	if err != nil {
		return false
	}

	builderRunImageRef, err := name.ParseReference(runImage)
	if err != nil {
		return false
	}

	return lastBuildRunImageRef.Identifier() == builderRunImageRef.Identifier()
}

func (b *Build) builtWithBuildpacks(buildpacks v1alpha1.BuildpackMetadataList) bool {
	for _, bp := range b.Status.BuildMetadata {
		if !buildpacks.Include(bp) {
			return false
		}
	}

	return true
}

func (b *Build) additionalBuildNeeded() bool {
	_, ok := b.Annotations[BuildNeededAnnotation]
	return ok
}
