package action

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Dreamacro/go-check/executor"

	"github.com/AlecAivazis/survey/v2"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

func Upgrade(cmd *cobra.Command, args []string) {
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Color("yellow")

	pwd, err := os.Getwd()
	if err != nil {
		println(err.Error())
		return
	}

	s.Suffix = " Running `go list -u -m -json all`"
	s.Start()
	output, err := executor.Exec(pwd)
	s.Stop()
	if err != nil {
		println(err.Error())
		return
	}

	list := executor.Scan([]byte(output))
	texts := []string{}
	mapping := map[string]*executor.Package{}
	for _, pkg := range list {
		if !pkg.Main && pkg.Update != nil && !pkg.Indirect {
			key := fmt.Sprintf("%s (%s --> %s)", pkg.Path, pkg.Version, pkg.Update.Version)
			texts = append(texts, key)
			mapping[key] = pkg
		}
	}

	if len(texts) == 0 {
		println("🎉  Your modules look amazing. Keep up the great work.")
		return
	}

	selected := []string{}
	prompt := &survey.MultiSelect{
		Message:  "Select the packages you want to upgrade",
		PageSize: 20,
		Options:  texts,
	}
	survey.AskOne(prompt, &selected)

	if len(selected) == 0 {
		return
	}

	answer := "\n\n" + strings.Join(selected, "\n") + "\n"
	prompt.Render(
		survey.MultiSelectQuestionTemplate,
		survey.MultiSelectTemplateData{
			MultiSelect: *prompt,
			Answer:      answer,
			ShowAnswer:  true,
			Config:      &survey.PromptConfig{},
		},
	)

	shouldUpgrade := []*executor.Package{}
	for _, item := range selected {
		shouldUpgrade = append(shouldUpgrade, mapping[item])
	}

	s.Suffix = " Installing using `go get`..."
	s.Start()
	_, err = executor.Upgrade(pwd, shouldUpgrade)
	s.Stop()
	if err != nil {
		println(err.Error())
		return
	}

	s.Suffix = " Running `go mod tidy`"
	s.Start()
	_, err = executor.Tidy(pwd)
	s.Stop()
	if err != nil {
		println(err.Error())
		return
	}

	println("🎉  Update complete!")
}
