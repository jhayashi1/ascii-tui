package tui

import (
	"fmt"
	"strconv"
	"strings"
)

// renderDetail builds the detail column's interior: labeled metadata
// rows for the selected entry, then a FILE section with on-disk facts.
// The surrounding renderColumn pads/clips rows to the column box.
func renderDetail(meta entryMeta, width int, st styles) string {
	rows := []string{
		kvRow("name", meta.name, width, st),
		kvRow("size", fmt.Sprintf("%dx%d", meta.width, meta.height), width, st),
		kvRow("frames", strconv.Itoa(meta.frames), width, st),
		kvRow("length", formatDuration(meta.length), width, st),
		kvRow("source", meta.source, width, st),
		kvRow("filter", onOff(meta.filter), width, st),
	}
	if meta.ramp != "" {
		rows = append(rows, kvRow("ramp", meta.ramp, width, st))
	}

	rows = append(rows, "", sectionRule("file", width, st))
	rows = append(rows, pathRows(meta.path, width, 3, st)...)
	bytesText, modifiedText := "-", "-"
	if meta.fileSize > 0 {
		bytesText = formatBytes(meta.fileSize)
	}
	if !meta.modTime.IsZero() {
		modifiedText = meta.modTime.Format("2006-01-02 15:04")
	}
	rows = append(rows,
		kvRow("bytes", bytesText, width, st),
		kvRow("modified", modifiedText, width, st),
	)
	return strings.Join(rows, "\n")
}

// pathRows renders the entry's on-disk path as at most maxLines rows: a
// dim "path" label prefixes the first line, with the path itself in
// normal text hard-wrapped across the rest; anything longer is cut.
func pathRows(s string, width, maxLines int, st styles) []string {
	const label = "path "
	if width <= len(label) || maxLines < 1 {
		return nil
	}
	first := truncateLabel(s, width-len(label))
	if first == "" {
		return nil
	}
	rows := []string{st.metaKey.Render(label) + st.metaValue.Render(first)}
	s = s[len(first):]
	for s != "" && len(rows) < maxLines {
		line := truncateLabel(s, width)
		if line == "" {
			break
		}
		rows = append(rows, st.metaValue.Render(line))
		s = s[len(line):]
	}
	return rows
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
