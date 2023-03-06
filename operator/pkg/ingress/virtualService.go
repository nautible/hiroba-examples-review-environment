package ingress

import (
	"context"
	"fmt"

	reviewv1alpha1 "github.com/nautible/review-env-operator/api/v1alpha1"
	networkingv1beta1 "istio.io/api/networking/v1beta1"
	istioclient "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type VirtualService struct {
	reviewv1alpha1.MergeRequest
}

func NewVirtualService(mr *reviewv1alpha1.MergeRequest) *VirtualService {
	return &VirtualService{*mr}
}

func (p *VirtualService) Create(ctx context.Context, client client.Client, name string) error {
	logger := log.FromContext(ctx)
	logger.Info("Create VirtualSerivce name : " + name)

	app := p.makeApp(name, p.Spec.Name, p.Spec.Application, p.Spec.TargetRevision)
	err := client.Create(ctx, app)
	if err != nil {
		logger.Error(err, "Check if the VirtualSerivce already exists, if not create a new one. Failed to create new VirtualSerivce", "VirtualSerivce", app.Name)
		return err
	}
	return nil
}

func (p *VirtualService) Delete(ctx context.Context, client client.Client, found *istioclient.VirtualService) error {
	logger := log.FromContext(ctx)
	logger.Info("Delete VirtualSerivce name : " + found.Name)

	err := client.Delete(ctx, found)
	if err != nil {
		logger.Error(err, "Check if the VirtualService delete error", "VirtualService", found.Name)
		return err
	}

	return nil
}

func (p *VirtualService) makeApp(name, groupName string, applicationName string, branch string) *istioclient.VirtualService {
	hosts := []string{"*"}
	gateways := []string{"application-gateway"}
	app := &istioclient.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: groupName,
		},
		Spec: networkingv1beta1.VirtualService{
			Gateways: gateways,
			Hosts:    hosts,
			Http:     httpRoute(groupName, applicationName, branch),
		},
	}
	return app
}

func httpRoute(groupName string, applicationName string, branch string) []*networkingv1beta1.HTTPRoute {
	// spec.http
	res := &networkingv1beta1.HTTPRoute{
		Name:  branch,
		Match: match(branch),
		Route: route(fmt.Sprintf("%s-%s", applicationName, branch)),
	}
	return []*networkingv1beta1.HTTPRoute{res}
}

func match(branch string) []*networkingv1beta1.HTTPMatchRequest {
	// spec.http.match
	param := &networkingv1beta1.StringMatch{
		MatchType: &networkingv1beta1.StringMatch_Exact{
			Exact: branch,
		},
	}
	res := &networkingv1beta1.HTTPMatchRequest{
		QueryParams: map[string]*networkingv1beta1.StringMatch{
			"branch": param,
		},
	}
	return []*networkingv1beta1.HTTPMatchRequest{res}
}

func route(name string) []*networkingv1beta1.HTTPRouteDestination {
	// spec.http.route
	res := &networkingv1beta1.HTTPRouteDestination{
		Destination: &networkingv1beta1.Destination{
			Host: name,
			Port: &networkingv1beta1.PortSelector{
				Number: 8080,
			},
		},
	}
	return []*networkingv1beta1.HTTPRouteDestination{res}
}
