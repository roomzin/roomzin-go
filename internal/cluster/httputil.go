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

var ErrNoLeaderAvailable = errors.New("no leader found in cluster")

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
	nodes := make(map[string]nodeInfo)

	existing := make(map[string]bool, len(hosts))
	for _, p := range hosts {
		existing[p] = true
	}
	discovered := make(map[string]bool) // node > is new

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
			nodes[host] = nodeInfo{
				host:      host,
				health:    health,
				leaderURL: info.LeaderURL,
			}
			mu.Unlock()

			var peers []string
			err := httpGet(host, cfg.APIPort, "/peers", cfg.AuthToken, cfg.HttpTimeout, &peers)
			if err != nil {
				return
			}
			for _, peer := range peers {
				mu.Lock()
				if _, ok := existing[peer]; !ok {
					discovered[peer] = true
				}
				mu.Unlock()
			}
		}(h)
	}
	wg.Wait()

	// Second phase: check new nodes
	var newWg sync.WaitGroup
	for host := range discovered {
		newWg.Add(1)
		go func(host string) {
			defer newWg.Done()

			health, e := healthCheck(host, cfg.APIPort, cfg.AuthToken, cfg.HttpTimeout)
			if e != nil || health == "unavailable" {
				return
			}

			info, e := getNodeInfo(host, cfg.APIPort, cfg.AuthToken, cfg.HttpTimeout)
			if e != nil {
				return
			}

			mu.Lock()
			nodes[host] = nodeInfo{
				host:      host,
				health:    health,
				leaderURL: info.LeaderURL,
			}
			mu.Unlock()
		}(host)
	}
	newWg.Wait()

	// Third phase: determine leader and followers
	votes := make(map[string]int, len(nodes))
	// First pass: count all votes
	for _, node := range nodes {
		if node.leaderURL != "" {
			votes[node.leaderURL]++
		}
	}

	// Find the leader URL with most votes
	var leaderURL string
	maxVotes := 0
	for url, count := range votes {
		if count > maxVotes {
			maxVotes = count
			leaderURL = url
		}
	}

	if leaderURL == "" {
		return "", nil, ErrNoLeaderAvailable
	}

	// Find the actual leader host and trusted followers
	var leader string
	var followers []string

	for _, node := range nodes {
		if node.leaderURL == leaderURL {
			switch node.health {
			case "active_leader":
				leader = node.host
			case "active_follower":
				followers = append(followers, node.host)
			}
		}
	}

	if leader == "" {
		return "", nil, ErrNoLeaderAvailable
	}

	return leader, followers, nil
}
