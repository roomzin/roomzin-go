package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var ErrNoLeaderAvailable = errors.New("no leader found in seed list")

type NodeInfo struct {
	NodeID    string `json:"node_id"`
	ZoneID    string `json:"zone_id"`
	ShardID   string `json:"shard_id"`
	LeaderID  string `json:"leader_id"`
	LeaderURL string `json:"leader_url"`
}

func parseHosts(s string) []string {
	out := strings.Split(s, ",")
	for i := 0; i < len(out); {
		out[i] = strings.TrimSpace(out[i])
		if out[i] == "" {
			out = append(out[:i], out[i+1:]...)
			continue
		}
		i++
	}
	return out
}

func httpGet(host string, port int, path string, authToken string, timeout time.Duration, dst any) error {
	url := fmt.Sprintf("http://%s:%d%s", host, port, path)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("http %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func getNodeInfo(host string, port int, authToken string, timeout time.Duration) (NodeInfo, error) {
	var out NodeInfo
	err := httpGet(host, port, "/node-info", authToken, timeout, &out)
	return out, err
}

func healthCheck(host string, port int, authToken string, timeout time.Duration) (string, error) {
	url := fmt.Sprintf("http://%s:%d/healthz", host, port)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("healthz %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func getClusterInfo(cfg *Config) (string, []string, error) {
	hosts := parseHosts(cfg.SeedHosts)

	type nodeInfo struct {
		host      string
		health    string
		leaderURL string
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	nodes := make([]nodeInfo, 0, len(hosts))

	// First phase: collect all node information
	for _, h := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()

			health, e := healthCheck(host, cfg.APIPort, cfg.AuthToken, cfg.HttpTimeout)
			if e != nil || health == "unavailable" {
				return
			}

			info, e := getNodeInfo(host, cfg.APIPort, cfg.AuthToken, cfg.HttpTimeout)
			if e != nil {
				return
			}

			mu.Lock()
			nodes = append(nodes, nodeInfo{
				host:      host,
				health:    health,
				leaderURL: info.LeaderURL,
			})
			mu.Unlock()
		}(h)
	}
	wg.Wait()

	// Second phase: determine leader and followers
	var leader string
	var followers []string

	for _, node := range nodes {
		switch node.health {
		case "active_leader":
			leader = node.host
		case "active_follower":
			followers = append(followers, node.host)
		}
	}

	if leader == "" {
		return "", nil, errors.New("no active leader found")
	}

	return leader, followers, nil
}
