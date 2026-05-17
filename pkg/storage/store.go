package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
	"github.com/weijia/my-k8s/pkg/api"
)

// Store 存储接口
type Store struct {
	db *sql.DB
}

// NewStore 创建存储
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	store := &Store{db: db}
	if err := store.initTables(); err != nil {
		return nil, err
	}

	return store, nil
}

// initTables 初始化表
func (s *Store) initTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		kind TEXT NOT NULL,
		namespace TEXT,
		name TEXT NOT NULL,
		uid TEXT UNIQUE NOT NULL,
		data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(namespace, name, kind)
	);
	CREATE INDEX IF NOT EXISTS idx_kind ON resources(kind);
	CREATE INDEX IF NOT EXISTS idx_namespace ON resources(namespace);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SavePod 保存 Pod
func (s *Store) SavePod(pod *api.Pod) error {
	if pod.UID == "" {
		pod.UID = fmt.Sprintf("%s-%d", pod.Name, time.Now().UnixNano())
	}
	if pod.CreationTimestamp.IsZero() {
		pod.CreationTimestamp = time.Now()
	}

	data, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		`INSERT INTO resources (kind, namespace, name, uid, data, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(namespace, name, kind) DO UPDATE SET
		data = excluded.data, updated_at = excluded.updated_at`,
		"Pod", pod.Namespace, pod.Name, pod.UID, string(data), time.Now(),
	)
	return err
}

// GetPod 获取 Pod
func (s *Store) GetPod(namespace, name string) (*api.Pod, error) {
	var data string
	err := s.db.QueryRow(
		"SELECT data FROM resources WHERE kind = ? AND namespace = ? AND name = ?",
		"Pod", namespace, name,
	).Scan(&data)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pod %s/%s not found", namespace, name)
	}
	if err != nil {
		return nil, err
	}

	var pod api.Pod
	if err := json.Unmarshal([]byte(data), &pod); err != nil {
		return nil, err
	}

	return &pod, nil
}

// ListPods 列出 Pod
func (s *Store) ListPods(namespace string) ([]api.Pod, error) {
	var rows *sql.Rows
	var err error

	if namespace == "" {
		rows, err = s.db.Query("SELECT data FROM resources WHERE kind = ?", "Pod")
	} else {
		rows, err = s.db.Query("SELECT data FROM resources WHERE kind = ? AND namespace = ?", "Pod", namespace)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pods []api.Pod
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var pod api.Pod
		if err := json.Unmarshal([]byte(data), &pod); err != nil {
			continue
		}
		pods = append(pods, pod)
	}

	return pods, nil
}

// DeletePod 删除 Pod
func (s *Store) DeletePod(namespace, name string) error {
	_, err := s.db.Exec(
		"DELETE FROM resources WHERE kind = ? AND namespace = ? AND name = ?",
		"Pod", namespace, name,
	)
	return err
}

// SaveNode 保存 Node
func (s *Store) SaveNode(node *api.Node) error {
	if node.UID == "" {
		node.UID = fmt.Sprintf("%s-%d", node.Name, time.Now().UnixNano())
	}
	if node.CreationTimestamp.IsZero() {
		node.CreationTimestamp = time.Now()
	}

	data, err := json.Marshal(node)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		`INSERT INTO resources (kind, namespace, name, uid, data, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(namespace, name, kind) DO UPDATE SET
		data = excluded.data, updated_at = excluded.updated_at`,
		"Node", "", node.Name, node.UID, string(data), time.Now(),
	)
	return err
}

// GetNode 获取 Node
func (s *Store) GetNode(name string) (*api.Node, error) {
	var data string
	err := s.db.QueryRow(
		"SELECT data FROM resources WHERE kind = ? AND name = ?",
		"Node", name,
	).Scan(&data)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("node %s not found", name)
	}
	if err != nil {
		return nil, err
	}

	var node api.Node
	if err := json.Unmarshal([]byte(data), &node); err != nil {
		return nil, err
	}

	return &node, nil
}

// ListNodes 列出 Node
func (s *Store) ListNodes() ([]api.Node, error) {
	rows, err := s.db.Query("SELECT data FROM resources WHERE kind = ?", "Node")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []api.Node
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var node api.Node
		if err := json.Unmarshal([]byte(data), &node); err != nil {
			continue
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// Close 关闭存储
func (s *Store) Close() error {
	return s.db.Close()
}
