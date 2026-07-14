package main

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}

func WriteSkillTable(w io.Writer, rows []SkillStatus) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "STATUS\tNAME\tPATH")
	for _, row := range rows {
		status := "disabled"
		if row.Enabled {
			status = "enabled"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", status, row.Name, row.Path)
	}
	_ = tw.Flush()
}
