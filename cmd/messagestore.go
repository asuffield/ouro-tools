package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/asuffield/ouro-tools/pkg/messagestore"
)

var (
	from     string
	template string
	to       string
	all      bool
)

func printSummary(s *messagestore.Store) {
	h := sha256.New()
	s.Hash(h, true)
	fmt.Printf("content hash: %s\n", hex.EncodeToString(h.Sum(nil)))
	fmt.Printf("%s\n", s.Summary())
}

var messagestoreConvertCmd = &cobra.Command{
	Use:   "messagestore",
	Short: "convert to/from the messagestore format",
	Long:  `Reads and writes messagestore text and binary files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if from == "" || to == "" {
			return fmt.Errorf("--from and --to are required")
		}

		s := messagestore.NewStore()
		s.Verbose = verbose
		s.BaseDir = filepath.Dir(from)

		if err := s.Read(from); err != nil {
			return fmt.Errorf("failed to read %s: %s", from, err)
		}

		if verbose {
			fmt.Printf("Input data:\n")
			printSummary(s)
		}

		var t *messagestore.Store
		if template != "" {
			t = messagestore.NewStore()
			t.Verbose = verbose
			t.BaseDir = filepath.Dir(from)
			if err := t.Read(template); err != nil {
				return fmt.Errorf("failed to read %s: %s", template, err)
			}
			if verbose {
				fmt.Printf("Template:\n")
				printSummary(t)
			}
		}

		return s.Write(to, t)
	},
}

func formatMessage(s *messagestore.Store, id string) string {
	var b strings.Builder
	b.WriteString(id)

	tys := s.MessageVarTypes(id)
	names := []string{}
	for name := range tys {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(&b, " {%s,%s}", name, tys[name])
	}
	fmt.Fprintf(&b, ": %s", s.Message(id))
	return b.String()
}

var messagestoreShowCmd = &cobra.Command{
	Use:   "messagestore <filename>",
	Short: "dump the messagestore format",
	Long:  `Reads messagestore text and binary files.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := messagestore.NewStore()
		s.Verbose = verbose

		if err := s.Read(args[0]); err != nil {
			return fmt.Errorf("failed to read %s: %s", args[0], err)
		}

		if verbose {
			fmt.Printf("Input data:\n")
			printSummary(s)
		}

		if all {
			args = s.MessageIDs()
			sort.Strings(args)
		}
		for _, id := range args {
			fmt.Printf("%s\n", formatMessage(s, id))
		}

		return nil
	},
}

var messagestoreDiffCmd = &cobra.Command{
	Use:   "messagestore <a> <b>",
	Short: "diff two files in the messagestore format",
	Long:  `Diffs messagestore files.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a := messagestore.NewStore()
		b := messagestore.NewStore()
		a.Verbose = verbose
		b.Verbose = verbose
		if err := a.Read(args[0]); err != nil {
			return fmt.Errorf("failed to read %s: %s", args[0], err)
		}
		if err := b.Read(args[1]); err != nil {
			return fmt.Errorf("failed to read %s: %s", args[1], err)
		}

		if verbose {
			fmt.Printf("Input data:\n")
			printSummary(a)
			printSummary(b)
		}

		idsA := a.MessageIDs()
		idsB := b.MessageIDs()
		sort.Strings(idsA)
		sort.Strings(idsB)

		var i, j int
		for i < len(idsA) && j < len(idsB) {
			if i > len(idsA) || idsA[i] > idsB[j] {
				fmt.Printf("-%s: %s\n", idsB[j], b.Message(idsB[j]))
				j += 1
			} else if j > len(idsB) || idsA[i] < idsB[j] {
				fmt.Printf("+%s: %s\n", idsA[i], a.Message(idsA[i]))
				i += 1
			} else {
				id := idsA[i]
				if id != idsB[j] {
					panic("bug in diff algorithm")
				}
				aMsg := formatMessage(a, id)
				bMsg := formatMessage(b, id)
				if aMsg != bMsg {
					fmt.Printf("-%s\n", aMsg)
					fmt.Printf("+%s\n", bMsg)
				}
				i += 1
				j += 1
			}
		}

		return nil
	},
}

func init() {
	convertCmd.AddCommand(messagestoreConvertCmd)

	messagestoreConvertCmd.Flags().StringVar(&from, "from", "", "file/directories to read from")
	messagestoreConvertCmd.Flags().StringVar(&template, "template", "", "file/directories to use as a template for writing")
	messagestoreConvertCmd.Flags().StringVar(&to, "to", "", "file/directory to write to")

	showCmd.AddCommand(messagestoreShowCmd)

	messagestoreShowCmd.Flags().BoolVar(&all, "all", false, "show all messages in store")

	diffCmd.AddCommand(messagestoreDiffCmd)
}
