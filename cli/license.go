package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func (c *CLI) setupLicenseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "license",
		Short: "Get license/feature codes",
		Long:  "Retrieve machine-bound license/feature codes based on physical network interfaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleLicense()
		},
	}
	return cmd
}

func (c *CLI) handleLicense() error {
	licenses, err := c.client.GetLicense()
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		return nil
	}

	if len(licenses) == 0 {
		fmt.Println("No valid network interfaces found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CODE\tINTERFACE\tIP")
	for _, l := range licenses {
		fmt.Fprintf(w, "%s\t%s\t%s\n", l.Code, l.Interface, l.IP)
	}
	w.Flush()
	return nil
}
