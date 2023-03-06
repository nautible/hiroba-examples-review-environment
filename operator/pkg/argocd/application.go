package argocd

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	reviewv1alpha1 "github.com/nautible/review-env-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ApplicationService struct {
	reviewv1alpha1.MergeRequest
}

func NewApplicationService(mr *reviewv1alpha1.MergeRequest) *ApplicationService {
	return &ApplicationService{*mr}
}

func (p *ApplicationService) Create(ctx context.Context, client client.Client, name string) error {
	logger := log.FromContext(ctx)
	logger.Info("Create Application name : " + name)
	finalizerName := "resources-finalizer.argocd.argoproj.io"
	app := p.createApp(name, p.Spec.Name, p.Spec.Application)
	if !controllerutil.ContainsFinalizer(app, finalizerName) {
		controllerutil.AddFinalizer(app, finalizerName)
	}
	err := client.Create(ctx, app)
	if err != nil {
		logger.Error(err, "Check if the Application already exists, if not create a new one. Failed to create new Application", "Application", app.Name)
		return err
	}
	return nil
}

func (p *ApplicationService) Delete(ctx context.Context, client client.Client, found *argocdv1alpha1.Application) error {
	logger := log.FromContext(ctx)
	logger.Info("4. Delete Application name : " + found.Name)

	err := client.Delete(ctx, found)
	if err != nil {
		logger.Error(err, "4. Check if the Application delete error", "Application", found.Name)
		return err
	}
	return nil
}

func (p *ApplicationService) createApp(name string, groupName string, applicationName string) *argocdv1alpha1.Application {
	app := &argocdv1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "argocd",
		},
		Spec: argocdv1alpha1.ApplicationSpec{
			Source:               source(p.Spec.BaseUrl, groupName, applicationName, p.Spec.ManifestPath, p.Spec.TargetRevision),
			Destination:          *destination(groupName, "https://kubernetes.default.svc", ""),
			Project:              "default",
			SyncPolicy:           syncPolicy(true, true, false), // いったん固定
			IgnoreDifferences:    nil,
			Info:                 nil,
			RevisionHistoryLimit: nil,
		},
	}
	return app
}

// リポジトリ指定のみサポート
func source(baseUrl string, groupName string, applicationName string, manifestPath string, targetRevision string) *argocdv1alpha1.ApplicationSource {
	repoURL := fmt.Sprintf("%s/%s/%s.git", baseUrl, groupName, applicationName)
	if manifestPath == "" {
		manifestPath = "/manifests/overlays/dev/" // default
	}
	if targetRevision == "" {
		targetRevision = "HEAD" // default
	}
	res := &argocdv1alpha1.ApplicationSource{
		RepoURL:        repoURL,
		Path:           manifestPath,
		TargetRevision: targetRevision,
		Helm:           nil,
		Kustomize:      nil,
		Directory:      nil,
		Plugin:         nil,
		Chart:          "",
	}
	return res
}

func destination(namespace string, server string, name string) *argocdv1alpha1.ApplicationDestination {
	res := &argocdv1alpha1.ApplicationDestination{
		Namespace: namespace,
		Server:    server,
		Name:      name,
	}
	return res
}

func syncPolicy(selfHeal bool, prune bool, allowEmpty bool) *argocdv1alpha1.SyncPolicy {
	res := &argocdv1alpha1.SyncPolicy{
		Automated: &argocdv1alpha1.SyncPolicyAutomated{
			SelfHeal:   selfHeal,
			Prune:      prune,
			AllowEmpty: allowEmpty,
		},
		SyncOptions: argocdv1alpha1.SyncOptions{},
		Retry:       &argocdv1alpha1.RetryStrategy{},
	}
	return res
}
