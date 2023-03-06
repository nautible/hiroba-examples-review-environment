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
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/nautible/review-env-operator/pkg/argocd"
	"github.com/nautible/review-env-operator/pkg/ingress"
	"github.com/nautible/review-env-operator/pkg/namespace"
	istioclient "istio.io/client-go/pkg/apis/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	reviewv1alpha1 "github.com/nautible/review-env-operator/api/v1alpha1"
)

// MergeRequestReconciler reconciles a MergeRequest object
type MergeRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=review.nautible.com,resources=mergerequests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=review.nautible.com,resources=mergerequests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=review.nautible.com,resources=mergerequests/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MergeRequest object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *MergeRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("0. start Reconcile mergerequest_controller")
	finalizerName := "mergerequest.review.nautible.com"

	// 1. MergeRequestリソースの取得
	mr := &reviewv1alpha1.MergeRequest{}
	err := r.Get(ctx, req.NamespacedName, mr)
	if apierrors.IsNotFound(err) {
		logger.Info("1. Fetch the MergeRequest instance. MergeRequest resource not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "1. Fetch the MergeRequest instance. Failed to get MergeRequest")
		return ctrl.Result{}, err
	}

	// 2. finalizer付与
	if !controllerutil.ContainsFinalizer(mr, finalizerName) {
		controllerutil.AddFinalizer(mr, finalizerName)
		if err = r.Update(ctx, mr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 9. deletion timestampがあれば関連リソースをすべて削除
	if !mr.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(mr, finalizerName) {
			r.delete(ctx, mr)
		}
		// // 関連リソース削除後にFinalizerを削除して更新（Finalizerがなくなったので次はカスタムリソース自体が削除される）
		controllerutil.RemoveFinalizer(mr, finalizerName)
		err = r.Update(ctx, mr)
		if err != nil {
			logger.Info("9. RemoveFinalizer Error name : " + mr.Spec.Name)
			return ctrl.Result{}, err
		}
		logger.Info("9. Delete Complete : " + mr.Spec.Name)
		return ctrl.Result{}, nil
	}

	// 3. MergeRequestリソースのnameに従いプロジェクト用のNamespaceを作成
	namespaceSvc := namespace.NewNameSpaceService(mr)
	namespaceSvc.CreateNamespace(ctx, r.Client)

	// グループ-プロジェクト-ブランチで名前を作る
	name := fmt.Sprintf("%s-%s-%s", mr.Spec.Name, mr.Spec.Application, strings.Replace(mr.Spec.TargetRevision, "/", "-", -1))

	// 4. Application作成
	applicationSvc := argocd.NewApplicationService(mr)
	applicationFound := &argocdv1alpha1.Application{}
	err = r.Get(ctx, types.NamespacedName{Name: name, Namespace: "argocd"}, applicationFound)
	if err != nil && apierrors.IsNotFound(err) {
		logger.Info("6. Application Create")
		applicationSvc.Create(ctx, r.Client, name)
	} else if err != nil {
		logger.Error(err, "6. Application Get Error")
		return ctrl.Result{}, err
	}

	// 5. Ingress(VirtualService)作成
	virtualServiceSvc := ingress.NewVirtualService(mr)
	virtualserviceFound := &istioclient.VirtualService{}
	err = r.Get(ctx, types.NamespacedName{Name: name, Namespace: mr.Spec.Name}, virtualserviceFound)
	if err != nil && apierrors.IsNotFound(err) {
		logger.Info("7. VirtualService Create")
		virtualServiceSvc.Create(ctx, r.Client, name)
	} else if err != nil {
		logger.Error(err, "7. VirtualService Get Error")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// 関連リソースの削除
func (r *MergeRequestReconciler) delete(ctx context.Context, mr *reviewv1alpha1.MergeRequest) {
	logger := log.FromContext(ctx)
	logger.Info("start delete")
	name := fmt.Sprintf("%s-%s-%s", mr.Spec.Name, mr.Spec.Application, strings.Replace(mr.Spec.TargetRevision, "/", "-", -1))

	virtualServiceSvc := ingress.NewVirtualService(mr)
	virtualserviceFound := &istioclient.VirtualService{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: mr.Spec.Name}, virtualserviceFound)
	if err != nil {
		logger.Error(err, "VirtualService delete error name : "+name)
	}
	virtualServiceSvc.Delete(ctx, r.Client, virtualserviceFound)

	applicationSvc := argocd.NewApplicationService(mr)
	applicationFound := &argocdv1alpha1.Application{}
	err = r.Get(ctx, types.NamespacedName{Name: name, Namespace: "argocd"}, applicationFound)
	if err != nil {
		logger.Error(err, "Application delete error name : "+name)
	}
	applicationSvc.Delete(ctx, r.Client, applicationFound)

	logger.Info("end delete")
}

// SetupWithManager sets up the controller with the Manager.
func (r *MergeRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&reviewv1alpha1.MergeRequest{}).
		Complete(r)
}
