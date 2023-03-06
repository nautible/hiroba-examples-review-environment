package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var config *rest.Config

type Client struct {
	clientset dynamic.Interface
}

type User struct {
	Id       int32  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type Project struct {
	Id                int32  `json:"id"`
	Name              string `json:"name"`
	WebUrl            string `json:"web_url"`
	Namespace         string `json:"namespace"`
	PathWithNamespace string `json:"path_with_namespace"`
	DefaultBranch     string `json:"default_branch"`
}
type ObjectAttributes struct {
	Title        string `json:"title"`
	Description  string `description:"description"`
	MergeStatus  string `json:"merge_status"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	State        string `json:"state"`
	Action       string `json:"action"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}
type MergeRequest struct {
	ObjectKind       string           `json:"object_kind"`
	EventType        string           `json:"event_type"`
	User             User             `json:"user"`
	Project          Project          `json:"project"`
	ObjectAttributes ObjectAttributes `json:"object_attributes"`
}

func main() {
	logger, err := NewLogger(os.Getenv("LOG_LEVEL"), os.Getenv("LOG_FORMAT"))
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		zap.S().Infow("healthz start")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Health Check OK")
	})
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		zap.S().Infow("webhook start Method : " + r.Method)
		c, err := NewClient()
		if err != nil {
			zap.S().Fatalw("InternalServerError")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "InternalServerError")
			return
		}
		switch r.Method {
		case http.MethodPost:
			token := r.Header.Get("X-Gitlab-Token")
			if !validToken(token) {
				zap.S().Warnw("AccessToken validation error")
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprintf(w, "Authorized Error")
				return
			}

			// MergeRequestの内容を取得
			body, _ := io.ReadAll(r.Body)
			defer r.Body.Close()
			zap.S().Debugw(string(body))
			var mergeRequest MergeRequest
			if err := json.Unmarshal(body, &mergeRequest); err != nil {
				zap.S().Errorw("json.Unmarshal error message : " + err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "InternalServerError")
				return
			}

			// マージ作成時およびマージ実施時以外のステータスは送信しない
			state := mergeRequest.ObjectAttributes.State
			action := mergeRequest.ObjectAttributes.Action
			if !checkStateAndAction(state, action) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "No Target Status.\n")
				return
			}

			// メッセージ送信
			target := mergeRequest.ObjectAttributes.SourceBranch
			application := mergeRequest.Project.Name
			group := mergeRequest.Project.Namespace
			if state == "opened" && action == "open" {
				zap.S().Infow("create MergeRequestResource")
				err = createCrd(r.Context(), c, group, application, target)
			} else {
				zap.S().Infow("delete MergeRequestResource")
				err = deleteCrd(r.Context(), c, group, application, target)
			}
			if err != nil {
				zap.S().Errorw("MergeRequestResource execute error message : " + err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "InternalServerError")
				return
			}
			zap.S().Infow("SendMessage Complete")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "SendMessage Complete")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, "Method not allowed.\n")
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// 環境変数 WEBHOOK_TOKEN に設定されているトークンとリクエストのトークンが一致しているか検証
func validToken(token string) bool {
	expect := os.Getenv("WEBHOOK_TOKEN")
	if expect == "" {
		return false
	}
	return token == expect
}

func checkStateAndAction(state string, action string) bool {
	if state == "opened" && action == "open" {
		// マージリクエスト作成時
		return true
	}
	if state == "merged" && action == "merge" {
		// マージリクエストマージ完了時
		return true
	}
	if state == "closed" && action == "close" {
		// マージリクエスト削除時
		return true
	}
	// 上記以外のWebhookは無視する
	return false
}

func NewClient() (client *Client, err error) {
	if config == nil {
		var kubeconfig string

		pathToConfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

		if exists(pathToConfig) {
			kubeconfig = pathToConfig
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		} else {
			config, err = rest.InClusterConfig()
		}
		if err != nil {
			return nil, err
		}
	}

	// create the clientset
	clientset, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		clientset: clientset,
	}, nil
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func createCrd(ctx context.Context, c *Client, groupName string, applicationName string, target string) error {
	if target == "" {
		return errors.New("target revision not found")
	}
	resource := schema.GroupVersionResource{Group: "review.nautible.com", Version: "v1alpha1", Resource: "mergerequests"}
	manifest := createManifest(groupName, applicationName, target)
	result, err := c.clientset.Resource(resource).Namespace("operator-system").Create(ctx, manifest, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("Created MergeRequest %q.\n", result.GetName())
	return nil
}

func deleteCrd(ctx context.Context, c *Client, groupName string, applicationName string, target string) error {
	if target == "" {
		return errors.New("target revision not found")
	}
	resource := schema.GroupVersionResource{Group: "review.nautible.com", Version: "v1alpha1", Resource: "mergerequests"}
	name := fmt.Sprintf("%s-%s-%s", groupName, applicationName, strings.Replace(target, "/", "-", -1))
	err := c.clientset.Resource(resource).Namespace("operator-system").Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("Deleted MergeRequest %q.\n", name)
	return nil
}

func createManifest(group string, project string, target string) *unstructured.Unstructured {
	name := fmt.Sprintf("%s-%s-%s", group, project, strings.Replace(target, "/", "-", -1))
	projectResource := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "review.nautible.com/v1alpha1",
			"kind":       "MergeRequest",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": "operator-system",
			},
			"spec": map[string]interface{}{
				"name":           group,
				"application":    project,
				"baseUrl":        "http://gitlab-webservice-default.gitlab.svc.cluster.local:8181",
				"manifestPath":   "manifests",
				"targetRevision": target,
			},
		},
	}
	return projectResource
}

func NewLogger(logLevel string, logFormat string) (*zap.Logger, error) {
	if logLevel == "" {
		logLevel = "DEBUG"
	}
	level, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		panic(err)
	}
	if logFormat == "" {
		logFormat = "console"
	}
	config := zap.Config{
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		Level:            level,
		Encoding:         logFormat,
		EncoderConfig: zapcore.EncoderConfig{
			LevelKey:       "level",
			TimeKey:        "timestamp",
			CallerKey:      "caller",
			MessageKey:     "msg",
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	return logger, nil
}
