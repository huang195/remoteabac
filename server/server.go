package server

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/kubernetes/pkg/apis/authorization/v1beta1"
	"k8s.io/kubernetes/pkg/auth/authorizer"
	"k8s.io/kubernetes/pkg/auth/authorizer/abac"
	"k8s.io/kubernetes/pkg/auth/user"
)

// Global abac authorizor that can be reloaded based on
// receiving a SIGUSR1 signal
var auth authorizer.Authorizer

const (
	defaultAddress string = ":8444"
)

type RemoteABACServer struct {
	Address       string
	PolicyFile    string
	TLSCertFile   string
	TLSPrivateKey string
}

func NewServer() *RemoteABACServer {
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
				auth, _ = abac.NewFromFile(s.PolicyFile)
				log.Printf("Reloading policy file from %s\n", s.PolicyFile)
			}
		}
	}()

	var err error
	auth, err = abac.NewFromFile(s.PolicyFile)
	if err != nil {
		log.Fatalf("Error reading policy file from %s: %v", s.PolicyFile, err)
	}

	log.Printf("Starting server and listening on %s\n", s.Address)
	mux := http.NewServeMux()
	mux.HandleFunc("/authorize", authorize)
	http.ListenAndServeTLS(s.Address, s.TLSCertFile, s.TLSPrivateKey, mux)
}

func (s *RemoteABACServer) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&s.Address, "address", s.Address, "Address remote ABAC server listens on (ip:port or :port to listen to all interfaces).")
	fs.StringVar(&s.PolicyFile, "authorization-policy-file", s.PolicyFile, "Authorization policy file.")
	fs.StringVar(&s.TLSCertFile, "tls-cert-file", s.TLSCertFile, "File containing x509 Certificate for HTTPS.")
	fs.StringVar(&s.TLSPrivateKey, "tls-private-key-file", s.TLSPrivateKey, "File containing x509 private key matching --tls-cert-file.")
}
