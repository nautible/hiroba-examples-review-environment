package namespace

import (
	"context"

	reviewv1alpha1 "github.com/nautible/review-env-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NameSpaceService struct {
	reviewv1alpha1.MergeRequest
}

func NewNameSpaceService(mr *reviewv1alpha1.MergeRequest) *NameSpaceService {
	return &NameSpaceService{*mr}
}

func (p *NameSpaceService) CreateNamespace(ctx context.Context, r client.Client) error {
	name := p.Spec.Name
	logger := log.FromContext(ctx)
	logger.Info("Create Namespace name : " + name)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	var namespaceFound corev1.Namespace
	err := r.Get(ctx, client.ObjectKey{Name: name, Namespace: ""}, &namespaceFound)
	// 初めてアプリケーションをデプロイするときにネームスペースも作成
	if apierrors.IsNotFound(err) {
		if err := r.Create(ctx, ns); err != nil {
			logger.Error(err, "namespace create error, if not create a new one. Failed to create new Namespace", "Namespace", ns.Name)
			return err
		}
		return nil
	} else if err != nil {
		logger.Error(err, "Fetch the Namespace instance. Failed to fetch namespace")
		return err
	}
	logger.Info("Fetch the Namespace instance. found namespace")
	return nil
}
