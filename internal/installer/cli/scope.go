// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nao1215/markdown"
	"github.com/pterm/pterm"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/models"
	mdrender "github.com/seniorGolang/tg/v3/internal/markdown"
)

// HandleScopeUse обрабатывает команду scope use.
func (inst *Installer) HandleScopeUse(ctx context.Context, args []string) (err error) {

	if len(args) == 0 {
		err = errors.New(i18n.Msg("Scope name not specified"))
		return
	}

	name := args[0]
	if err = inst.scopeManager.UseScope(ctx, name); err != nil {
		return
	}
	return
}

// HandleScopeList обрабатывает команду scope list.
func (inst *Installer) HandleScopeList(ctx context.Context) (err error) {

	var scopes []models.ScopeInfo
	if scopes, err = inst.scopeManager.ListScopes(ctx); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to get list of scopes: %w"), err)
		return
	}

	if len(scopes) == 0 {
		pterm.Info.Println(i18n.Msg("No scopes found"))
		return
	}

	header := []string{i18n.Msg("Name"), i18n.Msg("Status"), i18n.Msg("Packages"), i18n.Msg("Manifests")}
	rows := make([][]string, 0, len(scopes))

	for _, scope := range scopes {
		var status string
		if scope.IsActive {
			status = i18n.Msg("active")
		} else {
			status = "-"
		}

		scopeName := scope.Name
		if scope.IsActive {
			scopeName += " ✔"
		}

		rows = append(rows, []string{
			scopeName,
			status,
			fmt.Sprintf("%d", scope.PackageCount),
			fmt.Sprintf("%d", scope.ManifestCount),
		})
	}

	if err = renderMarkdownTable(header, rows); err != nil {
		return
	}

	return
}

// HandleScopeDelete обрабатывает команду scope del.
func (inst *Installer) HandleScopeDelete(ctx context.Context, args []string, force bool) (err error) {

	if len(args) == 0 {
		err = errors.New(i18n.Msg("Scope name not specified"))
		return
	}

	name := args[0]
	if err = inst.scopeManager.DeleteScope(ctx, name, force); err != nil {
		return
	}
	return
}

// HandleScopeShow обрабатывает команду scope show.
func (inst *Installer) HandleScopeShow(ctx context.Context, args []string) (err error) {

	var scopeName string
	if len(args) > 0 {
		scopeName = args[0]
	} else {
		if scopeName, err = inst.scopeManager.GetCurrentScope(ctx); err != nil {
			err = fmt.Errorf(i18n.Msg("Failed to get current scope: %w"), err)
			return
		}
	}

	var config *models.ScopeConfig
	if config, err = inst.scopeManager.GetScopeConfig(ctx, scopeName); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to get scope configuration: %w"), err)
		return
	}

	header := []string{i18n.Msg("Property"), i18n.Msg("Value")}
	rows := [][]string{
		{i18n.Msg("Scope"), config.Name},
		{i18n.Msg("Install Prefix"), config.InstallPrefix},
		{i18n.Msg("Bin Dir"), config.BinDir},
		{i18n.Msg("Lib Dir"), config.LibDir},
		{i18n.Msg("Config Dir"), config.ConfigDir},
	}

	if err = renderMarkdownTable(header, rows); err != nil {
		return
	}

	return
}

func renderMarkdownTable(header []string, rows [][]string) (err error) {

	var buf bytes.Buffer
	if err = markdown.NewMarkdown(&buf).
		Table(markdown.TableSet{
			Header: header,
			Rows:   rows,
		}).
		Build(); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to build markdown table: %w"), err)
		return
	}

	markdownContent := buf.String()

	// Вычисляем ширину таблицы на основе содержимого
	tableWidth := mdrender.CalculateTableWidth(header, rows)

	// Рендерим с вычисленной шириной
	var rendered string
	if rendered, err = mdrender.RenderContent(markdownContent, mdrender.WithWidth(tableWidth)); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to render markdown table: %w"), err)
		return
	}

	// Обрезаем лишние переносы строк в начале и конце
	rendered = trimNewlines(rendered)

	fmt.Print(rendered)
	fmt.Println()
	fmt.Println()

	return
}

// trimNewlines обрезает лишние переносы строк в начале и конце строки.
func trimNewlines(s string) (trimmed string) {

	// Обрезаем все переносы строк в начале и конце
	trimmed = strings.Trim(s, "\n\r")

	return trimmed
}
