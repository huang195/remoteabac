package policy

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

type EtcdRW struct {
	Path   string
	Client etcd.Client
	Kapi   etcd.KeysAPI
}

var _ ReaderWriter = (*EtcdRW)(nil)

// path is a comma separated list of etcd servers with key
// e.g., http://<ip1>:<port>/key/to/file,http://<ip2>:<port/key/to/file
func NewEtcdRW(path string) (*EtcdRW, error) {
	log.Printf("Loading policy file from etcd: %s\n", path)

	serverList := []string{}
	filePath := ""

	re := regexp.MustCompile(`(http[s]?://[a-zA-Z0-9\.]+:[0-9]+)/(.+)`)
	locations := strings.Split(path, ",")
	for _, location := range locations {
		result := re.FindStringSubmatch(location)
		if result == nil || len(result) != 3 {
			return nil, fmt.Errorf("etcd location is not recognized%s", location)
		}

		serverList = append(serverList, result[1])
		if filePath == "" {
			filePath = result[2]
		} else if filePath != result[2] {
			return nil, fmt.Errorf("All etcd path should be the same, %s does not match others\n", result[2])
		}
	}

	filePath = "/" + filePath
	//log.Printf("serverList: %s, path: %s\n", serverList, filePath)

	cfg := etcd.Config{
		Endpoints:               serverList,
		Transport:               etcd.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}

	client, err := etcd.New(cfg)
	if err != nil {
		return nil, err
	}

	return &EtcdRW{
		Path:   filePath,
		Client: client,
		Kapi:   etcd.NewKeysAPI(client),
	}, nil
}

func (f *EtcdRW) Read() (string, error) {
	resp, err := f.Kapi.Get(context.Background(), f.Path, nil)
	if err != nil {
		return "", err
	}

	//log.Printf("GET %q = %q\n", resp.Node.Key, resp.Node.Value)
	return resp.Node.Value, nil
}

func (f *EtcdRW) Write(content string) error {
	_, err := f.Kapi.Set(context.Background(), f.Path, content, nil)
	if err != nil {
		return err
	}

	//log.Printf("SET %q = %q\n", f.Path, content)
	return nil
}
