package server

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/guptarohit/asciigraph"
	"github.com/spf13/cobra"

	"github.com/hetznercloud/cli/internal/cmd/base"
	"github.com/hetznercloud/cli/internal/cmd/cmpl"
	"github.com/hetznercloud/cli/internal/cmd/output"
	"github.com/hetznercloud/cli/internal/cmd/util"
	"github.com/hetznercloud/cli/internal/hcapi2"
	"github.com/hetznercloud/cli/internal/state"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var metricTypeStrings = []string{
	string(hcloud.ServerMetricCPU),
	string(hcloud.ServerMetricDisk),
	string(hcloud.ServerMetricNetwork),
}

var MetricsCmd = base.Cmd{
	BaseCobraCommand: func(client hcapi2.Client) *cobra.Command {
		cmd := &cobra.Command{
			Use:                   fmt.Sprintf("metrics [options] (--type <%s>)... <server>", strings.Join(metricTypeStrings, "|")),
			Short:                 "[ALPHA] Metrics from a Server",
			ValidArgsFunction:     cmpl.SuggestArgs(cmpl.SuggestCandidatesF(client.Server().Names)),
			TraverseChildren:      true,
			DisableFlagsInUseLine: true,
		}

		cmd.Flags().StringSlice("type", nil, "Types of metrics you want to show")
		_ = cmd.MarkFlagRequired("type")
		_ = cmd.RegisterFlagCompletionFunc("type", cmpl.SuggestCandidates(metricTypeStrings...))

		cmd.Flags().String("start", "", "ISO 8601 timestamp")
		cmd.Flags().String("end", "", "ISO 8601 timestamp")

		output.AddFlag(cmd, output.OptionJSON(), output.OptionYAML())
		return cmd
	},
	Run: func(s state.State, cmd *cobra.Command, args []string) error {
		outputFlags := output.FlagsForCommand(cmd)

		idOrName := args[0]
		server, _, err := s.Client().Server().Get(s, idOrName)
		if err != nil {
			return err
		}
		if server == nil {
			return fmt.Errorf("Server not found: %s", idOrName)
		}

		metricTypesStr, _ := cmd.Flags().GetStringSlice("type")
		var metricTypes []hcloud.ServerMetricType
		for _, t := range metricTypesStr {
			if slices.Contains(metricTypeStrings, t) {
				metricTypes = append(metricTypes, hcloud.ServerMetricType(t))
			} else {
				return fmt.Errorf("invalid metric type: %s", t)
			}
		}

		start, _ := cmd.Flags().GetString("start")
		startTime := time.Now().Add(-30 * time.Minute)
		if start != "" {
			startTime, err = time.Parse(time.RFC3339, start)
			if err != nil {
				return fmt.Errorf("start date has an invalid format. It should be ISO 8601, like: %s", time.Now().Format(time.RFC3339))
			}
		}

		end, _ := cmd.Flags().GetString("end")
		endTime := time.Now()
		if end != "" {
			endTime, err = time.Parse(time.RFC3339, end)
			if err != nil {
				return fmt.Errorf("end date has an invalid format. It should be ISO 8601, like: %s", time.Now().Format(time.RFC3339))
			}
		}

		m, resp, err := s.Client().Server().GetMetrics(s, server, hcloud.ServerGetMetricsOpts{
			Types: metricTypes,
			Start: startTime,
			End:   endTime,
		})
		if err != nil {
			return err
		}
		switch {
		case outputFlags.IsSet("json") || outputFlags.IsSet("yaml"):
			var schema map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&schema); err != nil {
				return err
			}
			if outputFlags.IsSet("json") {
				return util.DescribeJSON(cmd.OutOrStdout(), schema)
			}
			return util.DescribeYAML(cmd.OutOrStdout(), schema)
		default:
			var keys []string
			for k := range m.TimeSeries {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if len(m.TimeSeries[k]) == 0 {
					cmd.Printf("Currently there are no metrics available. Please try it again later.")
					return nil
				}
				cmd.Printf("Server: %s \t Metric: %s \t Start: %s \t End: %s\n", server.Name, k, m.Start.String(), m.End.String())
				var data []float64
				for _, m := range m.TimeSeries[k] {
					d, _ := strconv.ParseFloat(m.Value, 64)
					data = append(data, d)
				}
				graph := asciigraph.Plot(data, asciigraph.Height(20), asciigraph.Width(100))
				cmd.Println(graph)
				cmd.Printf("\n\n")
			}
		}
		return nil
	},
}
