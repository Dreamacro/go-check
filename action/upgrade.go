package action

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dreamacro/go-check/common/batch"
	"github.com/Dreamacro/go-check/executor"

	"github.com/avast/retry-go/v4"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

type mod module.Version

func (m mod) String() string {
	return m.Path + "@" + m.Version
}

func Contains[S ~[]E, E comparable](s S, v E) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}

func Upgrade(cmd *cobra.Command, args []string) {
	pwd, err := os.Getwd()
	if err != nil {
		println(err.Error())
		return
	}

	modFile := filepath.Join(pwd, "go.mod")
	modBuf, err := os.ReadFile(modFile)
	if err != nil {
		println("❌ get go.mod failed:", err.Error())
		return
	}
	f, err := modfile.Parse(modFile, modBuf, nil)
	if err != nil {
		println(err.Error())
		return
	}

	directoryModulePath := []string{}
	for _, require := range f.Replace {
		if modfile.IsDirectoryPath(require.New.Path) {
			directoryModulePath = append(directoryModulePath, require.Old.Path)
		}
	}

	mainModule := []mod{}
	for _, require := range f.Require {
		if require.Indirect {
			continue
		}

		if Contains(directoryModulePath, require.Mod.Path) {
			println("🚧", require.Mod.Path, "is replace as a directory path, skip it.")
			continue
		}

		mainModule = append(mainModule, mod(require.Mod))
	}

	println("🔍 find main module:\n")
	for _, m := range mainModule {
		println(m.String())
	}
	println("")

	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("227"))
	cyanStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))

	b, ctx := batch.New(context.Background(), batch.WithConcurrencyNum[*executor.Module](10))

	var (
		result map[string]batch.Result[*executor.Module]
		bErr   *batch.Error
	)

	go func() {
		for _, module := range mainModule {
			m := module
			b.Go(m.Path, func() (ret *executor.Module, err error) {
				err = retry.Do(
					func() error {
						info, err := executor.GetModuleUpdate(pwd, m.Path)
						if err == nil {
							ret = info
						}
						return err
					},
				)
				return
			})
		}

		result, bErr = b.WaitAndGetResult()
	}()

	spinner.New().
		Type(spinner.Jump).
		Title(yellowStyle.Render(" Find module information...")).
		Context(ctx).
		Run()

	if bErr != nil {
		println(bErr.Err.Error())
		return
	}

	texts := []huh.Option[string]{}
	mapping := map[string]*executor.Module{}
	for _, require := range f.Require {
		if require.Indirect {
			continue
		}

		value, exist := result[require.Mod.Path]
		if !exist {
			continue
		}

		pkg := value.Value
		if pkg.Update == nil {
			continue
		}

		if pkg.Update.Version != require.Mod.Version {
			key := fmt.Sprintf("%s (%s --> %s)", pkg.Path, require.Mod.Version, pkg.Update.Version)
			texts = append(texts, huh.NewOption(key, key))
			mapping[key] = pkg
		}
	}

	if len(texts) == 0 {
		println("🎉  Your modules look amazing. Keep up the great work.")
		return
	}

	lipgloss.DefaultRenderer().Output().ClearScreen()

	selected := []string{}
	err = huh.NewMultiSelect[string]().
		Options(texts...).
		Title(cyanStyle.Render("Select the packages you want to upgrade")).
		Value(&selected).
		WithTheme(huh.ThemeBase16()).
		Run()

	if err != nil {
		return
	}

	if len(selected) == 0 {
		return
	}

	lipgloss.DefaultRenderer().Output().ClearScreen()

	answer := strings.Join(selected, "\n") + "\n"
	println(cyanStyle.Render(answer))

	shouldUpgrade := []*executor.Module{}
	for _, item := range selected {
		shouldUpgrade = append(shouldUpgrade, mapping[item])
	}

	spinner.New().
		Type(spinner.Jump).
		Title(yellowStyle.Render(" Installing using `go get`...")).
		Action(func() {
			_, err = executor.Upgrade(pwd, shouldUpgrade)
		}).
		Run()

	if err != nil {
		println(err.Error())
		return
	}

	spinner.New().
		Type(spinner.Jump).
		Title(yellowStyle.Render(" Running `go mod tidy`")).
		Action(func() {
			_, err = executor.Tidy(pwd)
		}).
		Run()

	if err != nil {
		println(err.Error())
		return
	}

	println("🎉  Update complete!")
}
