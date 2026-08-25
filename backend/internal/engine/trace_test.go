// 探索时间轴回归测试：验证事件偏移允许同刻发生，但禁止时间倒退和捷径重复生效。
package engine

import "testing"

func TestValidateTraceRejectsOffsetRegression(t *testing.T) {
	err := ValidateTrace([]TraceEvent{
		{Sequence: 1, Type: TraceRunStarted, OffsetSec: 10},
		{Sequence: 2, Type: TraceNodeEntered, OffsetSec: 9},
	}, 10)
	if err == nil {
		t.Fatal("事件偏移倒退时应返回错误")
	}
}

func TestEvacShortcutKeepsTraceMonotonic(t *testing.T) {
	trace := []TraceEvent{{Sequence: 1, Type: TraceNodeEntered, OffsetSec: 60}}
	state := eventRunState{DurationSec: 60, Trace: &trace}
	if _, err := applyEventEffect(EventEffect{Type: "evac_shortcut", Value: 2}, &state); err != nil {
		t.Fatalf("应用撤离捷径: %v", err)
	}
	state.DurationSec += state.consumeNextMoveDuration(60)
	state.addTrace(TraceEventTriggered, state.DurationSec, "node_test", "", nil)
	if err := ValidateTrace(trace, state.DurationSec); err != nil {
		t.Fatalf("捷径产生同刻事件时不应破坏时间轴: %v", err)
	}
}

func TestEvacShortcutAppliesOnlyToNextMove(t *testing.T) {
	state := eventRunState{}
	if _, err := applyEventEffect(EventEffect{Type: "evac_shortcut", Value: 2}, &state); err != nil {
		t.Fatalf("应用撤离捷径: %v", err)
	}
	if got := state.consumeNextMoveDuration(180); got != 60 {
		t.Fatalf("下一段移动耗时 = %d，期望 60 秒", got)
	}
	if got := state.consumeNextMoveDuration(180); got != 180 {
		t.Fatalf("捷径重复影响后续移动，耗时 = %d", got)
	}
}
