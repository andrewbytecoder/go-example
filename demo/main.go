package main

import (
	"fmt"
)

// Pod 表示一个 Pod 及其直接依赖
type Pod struct {
	Name      string   // Pod 名称
	DependsOn []string // 依赖的其他 Pod 名称（只有一层）
}

// DetectCircularDependency 检测一组 Pod 是否存在循环依赖
func DetectCircularDependency(pods []Pod) error {
	// 构建邻接表表示的依赖图：key 是 pod 名，value 是它依赖的 pod 列表（出边）
	graph := make(map[string][]string)
	allNodes := make(map[string]bool)

	for _, pod := range pods {
		graph[pod.Name] = append([]string(nil), pod.DependsOn...)
		allNodes[pod.Name] = true
		for _, dep := range pod.DependsOn {
			allNodes[dep] = true
		}
	}

	// 状态标记：0=未访问，1=正在 DFS 中（递归栈中），2=已访问完成
	visited := make(map[string]int)
	var stack []string // 用于记录拓扑排序结果

	var dfs func(node string) error
	dfs = func(node string) error {
		if status := visited[node]; status == 1 {
			// 发现正在访问的节点，说明有环
			return fmt.Errorf("circular dependency detected involving: %s", node)
		}
		if status := visited[node]; status == 2 {
			// 已经完成访问，无需重复处理
			return nil
		}

		// 标记为正在访问
		visited[node] = 1
		stack = append(stack, node)

		// 访问所有依赖项（即当前节点指向的节点）
		for _, neighbor := range graph[node] {
			if !allNodes[neighbor] {
				// 依赖的 Pod 不存在
				return fmt.Errorf("pod %s depends on non-existent pod: %s", node, neighbor)
			}
			if err := dfs(neighbor); err != nil {
				return err
			}
		}

		// 当前节点访问完成
		visited[node] = 2
		// 因为是拓扑排序，完成后可以弹出（可选），但这里我们只关心是否有环
		stack = stack[:len(stack)-1]
		return nil
	}

	// 对所有节点进行 DFS（包括没有被依赖的孤立节点）
	for node := range allNodes {
		if visited[node] == 0 {
			if err := dfs(node); err != nil {
				return err
			}
		}
	}

	// 可选：输出一个合法的启动顺序（逆序的拓扑排序）
	// reverse(stack) 就是启动顺序（从无依赖的开始）
	fmt.Println("No circular dependency found.")
	fmt.Println("Recommended startup order:", getTopologicalOrder(graph, allNodes))
	return nil
}

// getTopologicalOrder 返回一个合法的启动顺序（从最基础的开始）
func getTopologicalOrder(graph map[string][]string, allNodes map[string]bool) []string {
	visited := make(map[string]bool)
	var order []string

	var dfs func(node string)
	dfs = func(node string) {
		if visited[node] {
			return
		}
		visited[node] = true
		// 先处理所有依赖项
		for _, dep := range graph[node] {
			// 注意：dep 是 node 所依赖的，所以 dep 必须先启动
			dfs(dep)
		}
		// 所有依赖处理完后，再启动自己
		order = append(order, node)
	}

	for node := range allNodes {
		if !visited[node] {
			dfs(node)
		}
	}

	return order
}

// 示例使用
func main() {
	// 示例1：存在循环依赖
	fmt.Println("=== Test Case 1: Circular Dependency ===")
	pods1 := []Pod{
		{Name: "pod-a", DependsOn: []string{"pod-b"}},
		{Name: "pod-b", DependsOn: []string{"pod-c"}},
		{Name: "pod-c", DependsOn: []string{"pod-a"}}, // 循环
	}
	if err := DetectCircularDependency(pods1); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// 示例2：正常依赖，无环
	fmt.Println("\n=== Test Case 2: Acyclic Dependency ===")
	pods2 := []Pod{
		{Name: "pod-a", DependsOn: []string{"pod-b"}},
		{Name: "pod-b", DependsOn: []string{"pod-c"}},
		{Name: "pod-c", DependsOn: []string{"pod-e"}},
		{Name: "pod-e", DependsOn: []string{""}},
		{Name: "pod-f", DependsOn: []string{""}},
		{Name: "pod-g", DependsOn: []string{""}},
		{Name: "pod-d", DependsOn: []string{"pod-c", "pod-a"}},
	}
	if err := DetectCircularDependency(pods2); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// 示例3：依赖不存在的 Pod
	fmt.Println("\n=== Test Case 3: Missing Dependency ===")
	pods3 := []Pod{
		{Name: "pod-a", DependsOn: []string{"pod-x"}},
	}
	if err := DetectCircularDependency(pods3); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
