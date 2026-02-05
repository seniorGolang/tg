// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/nao1215/markdown"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	mdrender "github.com/seniorGolang/tg/v3/internal/markdown"
)

const versionLdVar = "tgp/internal.Version"

func Run(ctx context.Context, p Params) (err error) {

	versionLd := p.VersionLdVar
	if versionLd == "" {
		versionLd = versionLdVar
	}

	version := p.Version
	if version == "" {
		version = resolveVersion(p.RootDir)
	}

	slog.Debug(i18n.Msg("build started"), "root", p.RootDir, "out", p.OutDir, "scope", p.ScopeName, "version", version)

	pluginsDir := filepath.Join(p.RootDir, "plugins")
	var entries []os.DirEntry
	if entries, err = os.ReadDir(pluginsDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(i18n.Msg("plugins directory not found: %s"), pluginsDir)
		}
		return
	}

	var pluginDirs []string
	for _, e := range entries {
		if e.IsDir() {
			pluginDirs = append(pluginDirs, e.Name())
		}
	}

	if len(pluginDirs) == 0 {
		slog.Warn(i18n.Msg("no plugin directories in plugins/"))
		return
	}

	if err = os.MkdirAll(p.OutDir, 0755); err != nil {
		return
	}

	if p.Clean {
		if cleanErr := cleanOutDir(p.OutDir); cleanErr != nil {
			return cleanErr
		}
		slog.Debug(i18n.Msg("cleaned output directory"), "path", p.OutDir)
	}

	versionDisplay := version
	if version != "" && !strings.HasPrefix(version, "v") {
		versionDisplay = "v" + version
	}
	slog.Info(i18n.Msg("Building plugins") + " [" + strings.Join(pluginDirs, " ") + "] " + versionDisplay + " ...")

	slog.Debug("compile started", "plugins", len(pluginDirs))
	if err = compileAll(ctx, p.RootDir, p.OutDir, version, pluginDirs, versionLd); err != nil {
		return
	}
	slog.Debug("compile done")

	slog.Debug("compress started", "plugins", len(pluginDirs))
	if err = compressAll(ctx, p.OutDir, pluginDirs); err != nil {
		return
	}
	slog.Debug("compress done")

	slog.Debug("extract metadata started", "plugins", len(pluginDirs))
	var built []builtPlugin
	if built, err = extractMetadata(ctx, p.OutDir, p.ScopeName, pluginDirs); err != nil {
		return
	}
	slog.Debug("extract metadata done", "built", len(built))

	var genPath string
	if genPath, err = generateManifest(p.OutDir, version, built); err != nil {
		return
	}
	slog.Debug("manifest generated", "path", genPath)

	overridePath := p.OverrideManifest
	if overridePath == "" {
		overridePath = filepath.Join(p.RootDir, "manifest.overrides.yml")
	}
	if !filepath.IsAbs(overridePath) && p.RootDir != "" {
		overridePath = filepath.Join(p.RootDir, overridePath)
	}

	if err = mergeAndWrite(genPath, overridePath, p.OutDir); err != nil {
		return
	}

	absOut, _ := filepath.Abs(p.OutDir)
	installCmd := "tg pkg add file://" + absOut + " --force"

	if p.OutWriter != nil {
		if err = writeMarkdownOutput(p.OutWriter, built, installCmd); err != nil {
			return
		}
	} else {
		for _, b := range built {
			sizeStr := formatSize(b.TgpPath)
			slog.Info("  " + b.Name + "  " + sizeStr + "  " + b.Checksum)
		}
		slog.Info(installCmd)
	}

	return
}

func writeMarkdownOutput(w io.Writer, built []builtPlugin, installCmd string) (err error) {

	header := []string{i18n.Msg("Name"), i18n.Msg("Size"), i18n.Msg("Checksum")}
	rows := make([][]string, 0, len(built))
	for _, b := range built {
		rows = append(rows, []string{b.Name, formatSize(b.TgpPath), b.Checksum})
	}

	var tableBuf bytes.Buffer
	if err = markdown.NewMarkdown(&tableBuf).
		Table(markdown.TableSet{Header: header, Rows: rows}).
		Build(); err != nil {
		return
	}

	tableWidth := mdrender.CalculateTableWidth(header, rows)
	tableMD := strings.TrimSpace(tableBuf.String())
	codeBlockMD := "```bash\n" + installCmd + "\n```"
	fullMD := tableMD + "\n\n" + codeBlockMD

	var rendered string
	if rendered, err = mdrender.RenderContent(fullMD, mdrender.WithWidth(tableWidth)); err != nil {
		return
	}

	_, err = fmt.Fprint(w, strings.Trim(rendered, "\n\r"), "\n\n")
	return
}

func cleanOutDir(outDir string) (err error) {

	var entries []os.DirEntry
	if entries, err = os.ReadDir(outDir); err != nil {
		return
	}

	for _, e := range entries {
		path := filepath.Join(outDir, e.Name())
		if err = os.RemoveAll(path); err != nil {
			return
		}
	}

	return
}
