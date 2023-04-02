package calm

import (
	"fmt"
	"path"
	"runtime"
	"strings"
)

type StackPrintOptions struct {
	Formatter func(frame *StackFrame) string
	TrimStack bool
}

type StackFrame struct {
	Func, File string
	Line       int
	PC         uintptr
}

var (
	FullPrint      = &StackPrintOptions{Formatter: _DefaultPrint, TrimStack: false}
	TrimFullPrint  = &StackPrintOptions{Formatter: _DefaultPrint, TrimStack: true}
	ShortPrint     = &StackPrintOptions{Formatter: _DefaultShortPrint, TrimStack: false}
	TrimShortPrint = &StackPrintOptions{Formatter: _DefaultShortPrint, TrimStack: true}
)

func _DefaultPrint(f *StackFrame) string {
	return fmt.Sprintf("[%016x] %s at (%d:%s)\n", f.PC, f.Func, f.Line, f.File)
}

func _DefaultShortPrint(f *StackFrame) string {
	_, name := path.Split(f.File)
	_, fun := path.Split(f.Func)
	return fmt.Sprintf("%s at (%d:%s)\n", fun, f.Line, name)
}

type _Slice struct {
	info  ErrorInfo
	trace []uintptr
}

func _Flip(s []_Slice) {
	head := 0
	tail := len(s) - 1
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
		case *sErrChain:
			result = append(result, _Slice{info: n.info, trace: n.trace})
			c = n.next
		case *sErrNode:
			result = append(result, _Slice{info: n.info, trace: n.trace})
			_Flip(result)
			return
		}
	}
}

func _PrintableErrorTag(info ErrorInfo) string {
	if reg, ok := _Reg.Load(info.TCode()); ok {
		return reg.(IErrType).ErrName(info.ECode())
	} else {
		return fmt.Sprintf("error[%d:%d]", info.TCode(), info.ECode())
	}
}

func _PrintTaggedClean(b *strings.Builder, info ErrorInfo) {
	b.WriteString(_PrintableErrorTag(info))
	clean := info.Clean()
	if clean == "" {
		clean = pErrDefaultMsgSafe(info.TCode(), info.ECode())
	}
	if clean != "" {
		b.WriteString(": ")
		b.WriteString(clean)
	}
}

func PrintCleans(error Error) string {
	var builder strings.Builder
	slices := _TraceError(error)
	_PrintTaggedClean(&builder, slices[len(slices)-1].info)
	builder.WriteString("\n")
	nestId := len(slices)
	for i, slice := range slices[:len(slices)-1] {
		builder.WriteString(fmt.Sprintf("\tFrom[%d]: ", nestId-i))
		_PrintTaggedClean(&builder, slice.info)
		builder.WriteString("\n")
	}
	return builder.String()
}

func _PrintTaggedDetail(b *strings.Builder, info ErrorInfo) {
	b.WriteString(_PrintableErrorTag(info))
	detail := info.Detail()
	if detail == "" {
		detail = pErrDefaultMsgSafe(info.TCode(), info.ECode())
	}
	if detail != "" {
		b.WriteString(": ")
		b.WriteString(detail)
	}
}

func PrintDetails(error Error, option *StackPrintOptions) string {
	var builder strings.Builder
	slices := _TraceError(error)
	_PrintTaggedDetail(&builder, slices[len(slices)-1].info)
	builder.WriteString("\n")
	nestId := len(slices)
	for i, slice := range slices[:len(slices)-1] {
		builder.WriteString(fmt.Sprintf("\tFrom[%d]: ", nestId-i))
		_PrintTaggedDetail(&builder, slice.info)
		builder.WriteString("\n")
	}
	if option != nil {
		frames := make([][]StackFrame, 0)
		for _, slice := range slices {
			if slice.trace != nil {
				frames = append(frames, _PC2Frame(slice.trace))
			}
		}
		if option.TrimStack {
			frames = append(frames, _PC2Frame(pWithTrace()))
		}
		if len(frames) > 0 {
			builder.WriteString("Backtrace:\n")
			segments := _CollapseFrames(frames)
			if option.TrimStack {
				segments = segments[:len(segments)-1]
			}
			_PrintSegmentedFrames(&builder, segments, option.Formatter)
		}
	}
	return builder.String()
}

func _PC2Frame(pc []uintptr) (result []StackFrame) {
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
			if strings.Contains(frame.Function, "calm.") {
				continue
			}
			// collect all the remaining stack from the caller
			skip = false
		}
		result = append(result, StackFrame{Func: frame.Function, File: frame.File, Line: frame.Line, PC: frame.PC})
	}
	return
}

type _Segment struct {
	Branch, Stem []StackFrame
}

func _CollapseFrames(all [][]StackFrame) (result []_Segment) {
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
	result[0] = _Segment{Branch: []StackFrame{}, Stem: all[0]}
	return
}

func _PrintSegmentedFrames(builder *strings.Builder, s []_Segment, apply func(frame *StackFrame) string) {
	nestId := len(s) // this is in reverse, the first segment has the highest nesting depth
	for i, segment := range s {
		depth := nestId - i
		for _, f := range segment.Branch {
			builder.WriteString(fmt.Sprintf("%d|+\t", depth))
			builder.WriteString(apply(&f))
		}
		for _, f := range segment.Stem {
			builder.WriteString(fmt.Sprintf("%d|", depth))
			builder.WriteString(apply(&f))
		}
	}
}
