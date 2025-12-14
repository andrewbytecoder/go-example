// 文件路径: main_test.go
package main

import (
	"fmt"
	"testing"
)

// BenchmarkDetectCircularDependency_NoCycle 测试无循环依赖情况下的性能
func BenchmarkDetectCircularDependency_NoCycle(b *testing.B) {
	// 创建大量的无循环依赖 Pods
	pods := generatePodsWithoutCycle(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := DetectCircularDependency(pods)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDetectCircularDependency_WithCycle 测试存在循环依赖情况下的性能
func BenchmarkDetectCircularDependency_WithCycle(b *testing.B) {
	// 创建大量带有循环依赖的 Pods
	pods := generatePodsWithCycle(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DetectCircularDependency(pods)
		// 不检查错误因为肯定会返回循环依赖错误
	}
}

// BenchmarkDetectCircularDependency_Complex 测试复杂依赖关系下的性能
func BenchmarkDetectCircularDependency_Complex(b *testing.B) {
	// 创建复杂的依赖关系图
	pods := generateComplexDependencies(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := DetectCircularDependency(pods)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// generatePodsWithoutCycle 生成指定数量的无循环依赖 Pods
func generatePodsWithoutCycle(n int) []Pod {
	pods := make([]Pod, n)
	for i := 0; i < n; i++ {
		dependsOn := []string{}
		if i > 0 {
			// 每个 Pod 依赖于前一个 Pod
			dependsOn = append(dependsOn, fmt.Sprintf("pod-%d", i-1))
		}

		pods[i] = Pod{
			Name:      fmt.Sprintf("pod-%d", i),
			DependsOn: dependsOn,
		}
	}
	return pods
}

// generatePodsWithCycle 生成指定数量的带循环依赖 Pods
func generatePodsWithCycle(n int) []Pod {
	pods := make([]Pod, n)
	for i := 0; i < n; i++ {
		nextIndex := (i + 1) % n
		pods[i] = Pod{
			Name:      fmt.Sprintf("pod-%d", i),
			DependsOn: []string{fmt.Sprintf("pod-%d", nextIndex)},
		}
	}
	return pods
}

// generateComplexDependencies 生成复杂的依赖关系
func generateComplexDependencies(n int) []Pod {
	pods := make([]Pod, n)
	for i := 0; i < n; i++ {
		dependsOn := []string{}
		// 每个 Pod 依赖于前面最多5个 Pod
		for j := 1; j <= 5 && i-j >= 0; j++ {
			dependsOn = append(dependsOn, fmt.Sprintf("pod-%d", i-j))
		}

		pods[i] = Pod{
			Name:      fmt.Sprintf("pod-%d", i),
			DependsOn: dependsOn,
		}
	}
	return pods
}
