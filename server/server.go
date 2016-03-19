package server

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
	"k8s.io/kubernetes/pkg/apis/authorization/v1beta1"
	"k8s.io/kubernetes/pkg/auth/authorizer"
	"k8s.io/kubernetes/pkg/auth/authorizer/abac"
	"k8s.io/kubernetes/pkg/auth/user"
)

// Global abac authorizor that can be reloaded based on
// receiving a SIGUSR1 signal or etcd node change
var auth authorizer.Authorizer

const (
	defaultAddress   string = ":8444"
	tmpFile                 = "/tmp/abac-policy"
	retryFailureTime        = 10 * time.Second
)

type RemoteABACServer struct {
	Address       string
	PolicyFile    string
	TLSCertFile   string
	TLSPrivateKey string
}

func New() *RemoteABACServer {
	return &RemoteABACServer{
		Address: defaultAddress,
	}
}

func authorize(w http.ResponseWriter, r *http.Request) {

	// Decode request data
	var req v1beta1.SubjectAccessReview
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// TODO: is this the right way to handle request data
		// we do not recognize?
		w.WriteHeader(http.StatusNoContent)
	}

	attribs := authorizer.AttributesRecord{}
	attribs.User = &user.DefaultInfo{Name: req.Spec.User}
	if req.Spec.ResourceAttributes != nil {
		attribs.Verb = req.Spec.ResourceAttributes.Verb
		attribs.Namespace = req.Spec.ResourceAttributes.Namespace
		attribs.APIGroup = req.Spec.ResourceAttributes.Group
		attribs.Resource = req.Spec.ResourceAttributes.Resource
		attribs.ResourceRequest = true
	} else if req.Spec.NonResourceAttributes != nil {
		attribs.Verb = req.Spec.NonResourceAttributes.Verb
		attribs.Path = req.Spec.NonResourceAttributes.Path
		attribs.ResourceRequest = false
	}

	// Check ABAC authorization policy
	ret := auth.Authorize(attribs)

	// Create response
	res := &v1beta1.SubjectAccessReview{}
	res.Kind = req.Kind
	res.APIVersion = req.APIVersion
	if ret != nil {
		log.Printf("deny access to %s\n", req.Spec.User)
		res.Status.Allowed = false
		res.Status.Reason = ret.Error()
	} else {
		log.Printf("allow access to %s\n", req.Spec.User)
		res.Status.Allowed = true
	}

	// Encode response
	if err := json.NewEncoder(w).Encode(res); err != nil {
		// TODO: is this the right way to handle request data we do not recognize?
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *RemoteABACServer) Run() {
	s.AddFlags(flag.CommandLine)
	flag.Parse()

	// Reload authorization policy upon receiving a SIGUSR1 signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1)

	go func() {
		log.Printf("Listens to SIGUSER1...\n")
		for {
			select {
			case <-sigs:
				policyFile := handlePolicyFile(s.PolicyFile)
				auth, _ = abac.NewFromFile(policyFile)
				log.Printf("Reloading policy file from %s\n", s.PolicyFile)
			}
		}
	}()

	var err error
	policyFile := handlePolicyFile(s.PolicyFile)
	auth, err = abac.NewFromFile(policyFile)
	if err != nil {
		log.Fatalf("Error reading policy file from %s: %v", s.PolicyFile, err)
	}

	log.Printf("Starting server and listening on %s\n", s.Address)
	mux := http.NewServeMux()
	mux.HandleFunc("/authorize", authorize)
	http.ListenAndServeTLS(s.Address, s.TLSCertFile, s.TLSPrivateKey, mux)
}

func handlePolicyFile(policyFile string) string {

	// policyFile can be in different formats to specify their storage medium
	// In the most common case, where it is stored as a local file, one can specify
	//
	// --authorization-policy-file=myfile
	//
	// In the case of etcd backend, one can specify
	//
	// --authorization-policy-file=etcd@http://10.10.0.1:2379/path/to/policy/file
	//
	// One can also specify multiple etcd backends, e.g.,
	//
	// --authorization-policy-file=etcd@http://10.10.0.1:2379/path/to/policy/file,\
	//   http://10.10.0.2:2379/path/to/policy/file,\
	//   http://10.10.0.3:2379/path/to/policy/file
	//

	arr := strings.Split(policyFile, "@")
	if len(arr) == 1 {
		// This is a local file
		log.Printf("Loading policy file from a local file: %s\n", policyFile)
		return policyFile
	} else if len(arr) > 2 {
		log.Fatalf("Policy file is not correctly specified: %s\n", policyFile)
	}

	storageType := strings.ToLower(arr[0])
	policyFile = arr[1]

	switch storageType {
	case "etcd":
		log.Printf("Loading policy file from etcd: %s\n", policyFile)

		serverList := []string{}
		path := ""

		re := regexp.MustCompile(`(http[s]?://[a-zA-Z0-9\.]+:[0-9]+)/(.+)`)
		locations := strings.Split(policyFile, ",")
		for _, location := range locations {
			result := re.FindStringSubmatch(location)
			if result == nil || len(result) != 3 {
				log.Fatalf("etcd location is not recognized: %s\n", location)
			}

			serverList = append(serverList, result[1])
			if path == "" {
				path = result[2]
			} else if path != result[2] {
				log.Fatalf("All etcd path should be the same, %s does not match others\n", result[2])
			}
		}

		path = "/" + path
		log.Printf("serverList: %s, path: %s\n", serverList, path)

		cfg := etcd.Config{
			Endpoints:               serverList,
			Transport:               etcd.DefaultTransport,
			HeaderTimeoutPerRequest: time.Second,
		}

		client, err := etcd.New(cfg)
		if err != nil {
			log.Fatalf("Failed to create etcd connection: %v\n", err)
		}

		kapi := etcd.NewKeysAPI(client)
		if resp, err := kapi.Get(context.Background(), path, nil); err != nil {
			log.Fatalf("Cannot GET %s from etcd server: %v\n", path, err)
		} else {
			//log.Printf("%q key has %q value\n", resp.Node.Key, resp.Node.Value)
			fileName := tmpFile
			ioutil.WriteFile(fileName, []byte(resp.Node.Value), 0644)

			go func() {
				watcher := kapi.Watcher(path, nil)
				for {
					resp, err := watcher.Next(context.Background())
					if err != nil {
						log.Printf("Encountered error while watching for etcd: %v\n", err)
						time.Sleep(retryFailureTime)
					} else {
						//log.Printf("%q key has %q value\n", resp.Node.Key, resp.Node.Value)
						fileName := tmpFile
						ioutil.WriteFile(fileName, []byte(resp.Node.Value), 0644)
						auth, _ = abac.NewFromFile(fileName)
						log.Printf("Reloading policy file from %s\n", fileName)
					}
				}
			}()

			return fileName
		}

	default:
		log.Fatalf("Storage type %s is not currently supported\n", storageType)
	}

	// Should not reach here
	return ""
}

func (s *RemoteABACServer) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&s.Address, "address", s.Address, "Address remote ABAC server listens on (ip:port or :port to listen to all interfaces).")
	fs.StringVar(&s.PolicyFile, "authorization-policy-file", s.PolicyFile, "Authorization policy file.")
	fs.StringVar(&s.TLSCertFile, "tls-cert-file", s.TLSCertFile, "File containing x509 Certificate for HTTPS.")
	fs.StringVar(&s.TLSPrivateKey, "tls-private-key-file", s.TLSPrivateKey, "File containing x509 private key matching --tls-cert-file.")
}
