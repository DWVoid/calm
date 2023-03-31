package calm

import (
	"fmt"
	"runtime"
	"strings"
)

type _Slice struct {
	info  ErrorInfo
	trace []uintptr
}

func _Flip(s []_Slice) {
	head := 0
	tail := len(s) - 2
	for head < tail {
		tmp := s[head]
		s[head] = s[tail]
		s[tail] = tmp
		head++
		tail--
	}
}

func _TraceError(e Error) (result []_Slice) {
	var c any = e
	result = make([]_Slice, 0)
	for {
		switch n := c.(type) {
		case *errChain:
			result = append(result, _Slice{info: n.info, trace: n.trace})
			c = n.next
		case *errNode:
			result = append(result, _Slice{info: n.info, trace: n.trace})
			_Flip(result)
			return
		}
	}
}

func PrintCleans(error Error) string {
	var builder strings.Builder
	slices := _TraceError(error)
	builder.WriteString(slices[len(slices)-1].info.Clean())
	builder.WriteString("\n")
	nestId := len(slices)
	for i, slice := range slices[:len(slices)-1] {
		builder.WriteString(fmt.Sprintf("\tFrom[%d]: ", nestId-i))
		builder.WriteString(slice.info.Clean())
		builder.WriteString("\n")
	}
	return builder.String()
}

func PrintDetails(error Error, withTrace bool) string {
	var builder strings.Builder
	slices := _TraceError(error)
	builder.WriteString(slices[len(slices)-1].info.Detail())
	builder.WriteString("\n")
	nestId := len(slices)
	for i, slice := range slices[:len(slices)-1] {
		builder.WriteString(fmt.Sprintf("\tFrom[%d]: ", nestId-i))
		builder.WriteString(slice.info.Detail())
		builder.WriteString("\n")
	}
	if withTrace {
		frames := make([][]_Frame, 0)
		for _, slice := range slices {
			if slice.trace != nil {
				frames = append(frames, _PC2Frame(slice.trace))
			}
		}
		if len(frames) > 0 {
			builder.WriteString("Backtrace:\n")
			_PrintSegmentedFrames(builder, _CollapseFrames(frames))
		}
	}
	return builder.String()
}

type _Frame struct {
	Func, File string
	Line       int
	PC         uintptr
}

func _PC2Frame(pc []uintptr) (result []_Frame) {
	frames := runtime.CallersFrames(pc)
	skip := true
	more := true
	var frame runtime.Frame
	for more {
		frame, more = frames.Next()
		if skip {
			// skip the runtime.Callers frame
			if frame.Function == "runtime.Callers" {
				continue
			}
			// skip all top internal frames from the calm package to reduce noise
			if strings.Contains(frame.File, "calm/") {
				continue
			}
			// collect all the remaining stack from the caller
			skip = false
		}
		result = append(result, _Frame{Func: frame.Function, File: frame.File, Line: frame.Line, PC: frame.PC})
	}
	return
}

type _Segment struct {
	Branch, Stem []_Frame
}

func _CollapseFrames(all [][]_Frame) (result []_Segment) {
	count := len(all)
	result = make([]_Segment, count)
	level := 1
	unfinished := count
	for unfinished > 1 {
		satisfy := true
		// compare one layer of frame across all segments
		for i := 1; i < unfinished; i++ {
			l := all[i-1]
			r := all[i]
			if (level > len(l)) || (level > len(r)) {
				satisfy = false
				break
			}
			lhs := &l[len(l)-level]
			rhs := &r[len(r)-level]
			if lhs.PC == rhs.PC {
				continue
			}
			if (lhs.Func != rhs.Func) || (lhs.Line != rhs.Line) {
				satisfy = false
				break
			}
		}
		if satisfy {
			level++ // try the next layer
		} else {
			// the maximum common item count across all segments
			maxCommon := level - 1
			unfinished--
			// slice maxCommon items off all items except the last one
			for i := 0; i < unfinished; i++ {
				all[i] = all[i][:len(all[i])-maxCommon]
			}
			// craft the result of the last item
			slice := all[unfinished]
			result[unfinished] = _Segment{Branch: slice[:len(slice)-maxCommon], Stem: slice[len(slice)-maxCommon:]}
		}
	}
	// treat the remaining as stem, set result
	result[0] = _Segment{Branch: []_Frame{}, Stem: all[0]}
	return
}

func _PrintSegmentedFrames(builder strings.Builder, s []_Segment) {
	line := builder.WriteString
	nestId := len(s) // this is in reverse, the first segment has the highest nesting depth
	for i, segment := range s {
		depth := nestId - i
		for _, f := range segment.Branch {
			_, _ = line(fmt.Sprintf("%d|+\t[%016x] %s at (%d:%s)\n", depth, f.PC, f.Func, f.Line, f.File))
		}
		for _, f := range segment.Stem {
			_, _ = line(fmt.Sprintf("%d|[%016x] %s at (%d:%s)\n", depth, f.PC, f.Func, f.Line, f.File))
		}
	}
}
