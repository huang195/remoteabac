package policy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	//"log"
	"strings"

	abac "k8s.io/kubernetes/pkg/apis/abac/v1beta1"
)

type Policy struct {
	PolicyFile string // local file or etcd@http://<ip>:<port/path/to/file
	User       string // e.g., alice
	Namespace  string // e.g., alice_namespace
	Privileged bool   // e.g., false
	ReadOnly   bool   // e.g., false
	ReqType    string // e.g., add, delete
	RW         ReaderWriter
}

func New() (*Policy, error) {
	p := &Policy{}
	p.AddFlags(flag.CommandLine)
	flag.Parse()

	if strings.ToLower(p.ReqType) != "show" {
		if p.User == "" {
			return nil, fmt.Errorf("Need to specify user\n")
		}

		if strings.ToLower(p.ReqType) == "add" && !p.Privileged && p.Namespace == "" {
			return nil, fmt.Errorf("Need to specify a namespace for a non-privileged user\n")
		}
	}

	arr := strings.Split(p.PolicyFile, "@")
	if len(arr) == 1 {
		var err error
		p.RW, err = NewFileRW(p.PolicyFile)
		if err != nil {
			return nil, err
		}
	} else if len(arr) > 2 {
		return nil, fmt.Errorf("Policy file is not correctly specified: %s\n", p.PolicyFile)
	} else {
		storageType := strings.ToLower(arr[0])
		policyFile := arr[1]

		switch storageType {
		case "etcd":
			var err error
			p.RW, err = NewEtcdRW(policyFile)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("Storage type %s is not currently supported\n", storageType)
		}
	}
	return p, nil
}

func (p *Policy) ProcessRequest() error {
	switch strings.ToLower(p.ReqType) {
	case "add":
		r, err := p.RW.Read()
		if err != nil {
			return err
		}
		r, err = p.addUser(r)
		if err != nil {
			return err
		}
		return p.RW.Write(r)
	case "delete":
		r, err := p.RW.Read()
		if err != nil {
			return err
		}
		r, err = p.deleteUser(r)
		if err != nil {
			return err
		}
		return p.RW.Write(r)
	case "show":
		r, err := p.RW.Read()
		if err != nil {
			return err
		}
		fmt.Printf("%v", r)
	}
	return fmt.Errorf("Request type not specified or not recognized: \"%s\"\n", p.ReqType)
}

// Assumes p.User does not exist in policy
func (p *Policy) addUser(policy string) (string, error) {
	ret := policy
	po := abac.Policy{}
	po.APIVersion = "abac.authorization.kubernetes.io/v1beta1"
	po.Kind = "Policy"
	po.Spec.User = p.User
	po.Spec.Namespace = p.Namespace
	po.Spec.Resource = "*"
	po.Spec.APIGroup = "*"
	if p.Privileged {
		po.Spec.Namespace = "*"
		po1 := abac.Policy{}
		po1.APIVersion = "abac.authorization.kubernetes.io/v1beta1"
		po1.Kind = "Policy"
		po1.Spec.User = p.User
		po1.Spec.NonResourcePath = "*"
		b := new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(po1); err != nil {
			return "", err
		}
		ret = ret + b.String()
	} else if p.ReadOnly {
		po.Spec.Readonly = true
	}
	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(po); err != nil {
		return "", err
	}
	ret = ret + b.String()
	//log.Printf("%v", ret)
	return ret, nil
}

func (p *Policy) deleteUser(policy string) (string, error) {
	var ret string
	scanner := bufio.NewScanner(strings.NewReader(policy))
	for scanner.Scan() {
		var po abac.Policy
		b := scanner.Bytes()

		// skip comment lines and blank lines
		trimmed := strings.TrimSpace(string(b))
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			ret = ret + string(b) + "\n"
			continue
		}

		if err := json.NewDecoder(strings.NewReader(trimmed)).Decode(&po); err != nil {
			return "", err
		}

		if po.Spec.User == p.User {
			continue
		} else {
			ret = ret + string(trimmed) + "\n"
		}
	}
	//log.Printf("%v", ret)
	return ret, nil
}

func (p *Policy) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&p.PolicyFile, "authorization-policy-file", p.PolicyFile, "Authorization policy file.")
	fs.StringVar(&p.User, "user", p.User, "User for the request.")
	fs.StringVar(&p.Namespace, "namespace", p.Namespace, "Namespace of the user.")
	fs.BoolVar(&p.Privileged, "privileged", p.Privileged, "Is user a privileged user")
	fs.BoolVar(&p.ReadOnly, "readonly", p.ReadOnly, "Does user have readonly access to the namespace")
	fs.StringVar(&p.ReqType, "type", p.ReqType, "Type of request: add, delete, show.")
}
