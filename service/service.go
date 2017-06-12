package service

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/aerokube/selenoid/config"
	"github.com/docker/docker/client"
)

// Starter - interface to create session with cancellation ability
type Starter interface {
	StartWithCancel() (*url.URL, string, string, func(), error)
}

// Manager - interface to choose appropriate starter
type Manager interface {
	Find(s string, v *string, sr string, vnc bool, requestId uint64) (Starter, bool)
}

// DefaultManager - struct for default implementation
type DefaultManager struct {
	IP       string
	InDocker bool
	CPU      int64
	Memory   int64
	Client   *client.Client
	Config   *config.Config
}

// Find - default implementation Manager interface
func (m *DefaultManager) Find(s string, v *string, sr string, vnc bool, requestId uint64) (Starter, bool) {
	log.Printf("[%d] [LOCATING_SERVICE] [%s-%s]\n", requestId, s, *v)
	service, ok := m.Config.Find(s, v)
	if !ok {
		return nil, false
	}
	switch service.Image.(type) {
	case string:
		if m.Client == nil {
			return nil, false
		}
		log.Printf("[%d] [USING_DOCKER] [%s-%s]\n", requestId, s, *v)
		return &Docker{m.IP, m.InDocker, m.CPU, m.Memory, m.Client, service, m.Config.ContainerLogs, sr, vnc, requestId}, true
	case []interface{}:
		log.Printf("[%d] [USING_DRIVER] [%s-%s]\n", requestId, s, *v)
		return &Driver{m.InDocker, service, requestId}, true
	}
	return nil, false
}

func wait(u string, t time.Duration) error {
	done := make(chan struct{})
	go func(done chan struct{}) {
	loop:
		for {
			select {
			case <-time.After(50 * time.Millisecond):
				req, _ := http.NewRequest(http.MethodHead, u, nil)
				req.Close = true
				r, err := http.DefaultClient.Do(req)
				if err == nil {
					r.Body.Close()
					done <- struct{}{}
				}
			case <-done:
				break loop
			}
		}
	}(done)
	select {
	case <-time.After(t):
		return fmt.Errorf("%s does not respond in %v", u, t)
	case <-done:
	}
	close(done)
	return nil
}
