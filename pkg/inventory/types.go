package inventory

// Host 表示一个主机
type Host struct {
	Name   string                 // Inventory hostname (alias)
	Vars   map[string]interface{} // 包含 ansible_host, ansible_port 等
	Groups []string               // 所属组名
}

// Group 表示一个主机组
type Group struct {
	Name     string
	Hosts    []string // 主机名列表
	Children []string // 子组名列表
	Vars     map[string]interface{}
	Parents  []string // 父组名列表 (用于计算变量优先级)
}

// Inventory 表示整个 inventory
type Inventory struct {
	Hosts  map[string]*Host
	Groups map[string]*Group
}

// NewInventory 创建一个新的 Inventory
func NewInventory() *Inventory {
	inv := &Inventory{
		Hosts:  make(map[string]*Host),
		Groups: make(map[string]*Group),
	}

	// 创建默认组
	inv.Groups["all"] = &Group{
		Name:     "all",
		Hosts:    []string{},
		Children: []string{},
		Vars:     make(map[string]interface{}),
		Parents:  []string{},
	}

	inv.Groups["ungrouped"] = &Group{
		Name:     "ungrouped",
		Hosts:    []string{},
		Children: []string{},
		Vars:     make(map[string]interface{}),
		Parents:  []string{"all"},
	}

	return inv
}
