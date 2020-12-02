// +build !ignore_autogenerated

/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProvisionedService) DeepCopyInto(out *ProvisionedService) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProvisionedService.
func (in *ProvisionedService) DeepCopy() *ProvisionedService {
	if in == nil {
		return nil
	}
	out := new(ProvisionedService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ProvisionedService) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProvisionedServiceSpec) DeepCopyInto(out *ProvisionedServiceSpec) {
	*out = *in
	out.Binding = in.Binding
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProvisionedServiceSpec.
func (in *ProvisionedServiceSpec) DeepCopy() *ProvisionedServiceSpec {
	if in == nil {
		return nil
	}
	out := new(ProvisionedServiceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProvisionedServiceStatus) DeepCopyInto(out *ProvisionedServiceStatus) {
	*out = *in
	in.Status.DeepCopyInto(&out.Status)
	out.Binding = in.Binding
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProvisionedServiceStatus.
func (in *ProvisionedServiceStatus) DeepCopy() *ProvisionedServiceStatus {
	if in == nil {
		return nil
	}
	out := new(ProvisionedServiceStatus)
	in.DeepCopyInto(out)
	return out
}
