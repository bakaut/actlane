package adoption

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/actlane/actlane/packages/cli/internal/pack"
)

const markerPrefix = "actlane"

type PlanOptions struct {
	Target      string
	From        string
	Project     string
	JSON        bool
	Diff        bool
	ShowContent bool
	Marker      string
}

type ApplyOptions struct {
	PlanOptions
	DryRun bool
}

type RemoveOptions struct {
	PlanOptions
	DryRun bool
}

type ApplyResult struct {
	Target     string           `json:"target"`
	Project    string           `json:"project"`
	Generated  string           `json:"generated"`
	DryRun     bool             `json:"dryRun"`
	Operations []ApplyOperation `json:"operations"`
	Conflicts  int              `json:"conflicts"`
}

type ApplyOperation struct {
	Action      string `json:"action"`
	TargetPath  string `json:"targetPath"`
	OwnershipID string `json:"ownershipId"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
	MarkerStyle string `json:"markerStyle,omitempty"`
}

type RemoveResult struct {
	Target     string            `json:"target"`
	Project    string            `json:"project"`
	DryRun     bool              `json:"dryRun"`
	Operations []RemoveOperation `json:"operations"`
	Conflicts  int               `json:"conflicts"`
}

type RemoveOperation struct {
	Action      string `json:"action"`
	TargetPath  string `json:"targetPath"`
	OwnershipID string `json:"ownershipId"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
	MarkerStyle string `json:"markerStyle,omitempty"`
}

type Plan struct {
	Target     string          `json:"target"`
	Project    string          `json:"project"`
	Generated  string          `json:"generated"`
	Operations []PlanOperation `json:"operations"`
	Conflicts  int             `json:"conflicts"`
}

type PlanOperation struct {
	Action         string      `json:"action"`
	TargetPath     string      `json:"targetPath"`
	GeneratedPath  string      `json:"generatedPath,omitempty"`
	OwnershipID    string      `json:"ownershipId"`
	Reason         string      `json:"reason"`
	Preview        PlanPreview `json:"preview"`
	Diff           string      `json:"diff,omitempty"`
	Content        string      `json:"content,omitempty"`
	DisplayContent string      `json:"-"`
	MarkerStyle    string      `json:"markerStyle,omitempty"`
}

type PlanPreview struct {
	Lines  int    `json:"lines"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

func BuildPlan(loaded *pack.LoadedPack, opts PlanOptions) (*Plan, error) {
	if opts.Target == "" {
		return nil, fmt.Errorf("target is required")
	}
	if opts.Project == "" {
		opts.Project = "."
	}
	if opts.Marker == "" {
		opts.Marker = markerPrefix
	}
	targetProfile, err := targetProfileForPlan(loaded, opts.Target)
	if err != nil {
		return nil, err
	}
	files, err := targetFilesForAdoption(targetProfile)
	if err != nil {
		return nil, err
	}
	if opts.From == "" {
		opts.From = filepath.Join(loaded.Root, filepath.FromSlash(targetProfile.Spec.Output.Root))
	}
	plan := &Plan{
		Target:    opts.Target,
		Project:   opts.Project,
		Generated: opts.From,
	}
	for _, file := range files {
		if file.TargetPath == "" || file.GeneratedPath == "" {
			continue
		}
		op, err := planTargetFile(loaded, targetProfile, file, opts)
		if err != nil {
			return nil, err
		}
		plan.Operations = append(plan.Operations, op)
		if op.Action == "conflict" {
			plan.Conflicts++
		}
	}
	sort.Slice(plan.Operations, func(i, j int) bool {
		if plan.Operations[i].Action == plan.Operations[j].Action {
			return plan.Operations[i].TargetPath < plan.Operations[j].TargetPath
		}
		return plan.Operations[i].Action < plan.Operations[j].Action
	})
	return plan, nil
}

func FormatPlanText(plan *Plan, opts PlanOptions) string {
	groups := []struct {
		action string
		title  string
	}{
		{action: "create_file", title: "Will create:"},
		{action: "append_owned_block", title: "Will append Actlane block:"},
		{action: "update_owned_block", title: "Will update Actlane block:"},
		{action: "update_owned_file", title: "Will update Actlane-owned files:"},
		{action: "skip_existing_user_file", title: "Will skip unchanged existing files:"},
		{action: "conflict", title: "Conflicts:"},
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Plan target: %s\n", plan.Target)
	fmt.Fprintf(&b, "Project: %s\n", plan.Project)
	fmt.Fprintf(&b, "Generated: %s\n", plan.Generated)
	for _, group := range groups {
		var ops []PlanOperation
		for _, op := range plan.Operations {
			if op.Action == group.action {
				ops = append(ops, op)
			}
		}
		if len(ops) == 0 {
			continue
		}
		b.WriteString("\n")
		b.WriteString(group.title)
		b.WriteString("\n")
		for _, op := range ops {
			fmt.Fprintf(&b, "- %s", op.TargetPath)
			if op.Reason != "" {
				fmt.Fprintf(&b, " (%s)", op.Reason)
			}
			b.WriteString("\n")
			if op.GeneratedPath != "" {
				fmt.Fprintf(&b, "  source: %s\n", op.GeneratedPath)
			}
			fmt.Fprintf(&b, "  ownership: %s\n", op.OwnershipID)
			fmt.Fprintf(&b, "  preview: %d lines, %d bytes, sha256:%s\n", op.Preview.Lines, op.Preview.Bytes, op.Preview.SHA256)
			if opts.Diff && op.Diff != "" {
				b.WriteString("  diff:\n")
				b.WriteString(formatFenced(op.Diff, "diff", false))
			}
			if opts.ShowContent && op.DisplayContent != "" {
				b.WriteString("  content:\n")
				b.WriteString(formatFenced(op.DisplayContent, "text", true))
			}
		}
	}
	if plan.Conflicts > 0 {
		fmt.Fprintf(&b, "\nApply blocked: %d conflict(s)\n", plan.Conflicts)
	}
	return b.String()
}

func FormatPlanJSON(plan *Plan, opts PlanOptions) ([]byte, error) {
	copyPlan := *plan
	copyPlan.Operations = make([]PlanOperation, len(plan.Operations))
	copy(copyPlan.Operations, plan.Operations)
	for i := range copyPlan.Operations {
		if !opts.Diff {
			copyPlan.Operations[i].Diff = ""
		}
		if !opts.ShowContent {
			copyPlan.Operations[i].Content = ""
		}
	}
	return json.MarshalIndent(copyPlan, "", "  ")
}

func ApplyPlan(plan *Plan, opts ApplyOptions) (*ApplyResult, error) {
	if opts.Marker == "" {
		opts.Marker = markerPrefix
	}
	result := &ApplyResult{
		Target:    plan.Target,
		Project:   plan.Project,
		Generated: plan.Generated,
		DryRun:    opts.DryRun,
		Conflicts: plan.Conflicts,
	}
	for _, op := range plan.Operations {
		result.Operations = append(result.Operations, ApplyOperation{
			Action:      op.Action,
			TargetPath:  op.TargetPath,
			OwnershipID: op.OwnershipID,
			Status:      applyStatus(op.Action, opts.DryRun),
			Reason:      op.Reason,
			MarkerStyle: op.MarkerStyle,
		})
	}
	if plan.Conflicts > 0 {
		return result, fmt.Errorf("apply blocked: %d conflict(s)", plan.Conflicts)
	}
	if opts.DryRun {
		return result, nil
	}
	for _, op := range plan.Operations {
		if err := applyOperation(plan.Project, opts.Marker, op); err != nil {
			return result, err
		}
	}
	return result, nil
}

func FormatApplyText(result *ApplyResult) string {
	groups := []struct {
		status string
		title  string
	}{
		{status: "created", title: "Created:"},
		{status: "appended", title: "Appended Actlane block:"},
		{status: "updated", title: "Updated Actlane block:"},
		{status: "skipped", title: "Skipped:"},
		{status: "planned", title: "Dry-run planned:"},
		{status: "conflict", title: "Conflicts:"},
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Apply target: %s\n", result.Target)
	fmt.Fprintf(&b, "Project: %s\n", result.Project)
	fmt.Fprintf(&b, "Generated: %s\n", result.Generated)
	if result.DryRun {
		b.WriteString("Mode: dry-run\n")
	}
	for _, group := range groups {
		var ops []ApplyOperation
		for _, op := range result.Operations {
			if op.Status == group.status {
				ops = append(ops, op)
			}
		}
		if len(ops) == 0 {
			continue
		}
		b.WriteString("\n")
		b.WriteString(group.title)
		b.WriteString("\n")
		for _, op := range ops {
			fmt.Fprintf(&b, "- %s", op.TargetPath)
			if op.Reason != "" {
				fmt.Fprintf(&b, " (%s)", op.Reason)
			}
			b.WriteString("\n")
		}
	}
	if result.Conflicts > 0 {
		fmt.Fprintf(&b, "\nApply blocked: %d conflict(s)\n", result.Conflicts)
	}
	return b.String()
}

func FormatApplyJSON(result *ApplyResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

func RemovePlan(loaded *pack.LoadedPack, opts RemoveOptions) (*RemoveResult, error) {
	if opts.Target == "" {
		return nil, fmt.Errorf("target is required")
	}
	if opts.Project == "" {
		opts.Project = "."
	}
	if opts.Marker == "" {
		opts.Marker = markerPrefix
	}
	targetProfile, err := targetProfileForPlan(loaded, opts.Target)
	if err != nil {
		return nil, err
	}
	files, err := targetFilesForAdoption(targetProfile)
	if err != nil {
		return nil, err
	}
	if opts.From == "" {
		opts.From = filepath.Join(loaded.Root, filepath.FromSlash(targetProfile.Spec.Output.Root))
	}
	result := &RemoveResult{Target: opts.Target, Project: opts.Project, DryRun: opts.DryRun}
	for _, file := range files {
		if file.TargetPath == "" {
			continue
		}
		targetRel, err := cleanPlanPath(file.TargetPath)
		if err != nil {
			return nil, err
		}
		generatedRel, err := generatedRelForPlan(targetProfile, file)
		if err != nil {
			return nil, err
		}
		sourcePath := filepath.Join(opts.From, filepath.FromSlash(generatedRel))
		ownershipID := ownershipID(loaded.Manifest.Metadata.Name, targetRel)
		op := planRemoveOperation(opts.Project, opts.Marker, sourcePath, targetRel, ownershipID, file)
		if opts.DryRun && op.Status != "conflict" && op.Status != "missing" {
			op.Status = "planned"
		}
		result.Operations = append(result.Operations, op)
		if op.Status == "conflict" {
			result.Conflicts++
		}
	}
	if result.Conflicts > 0 {
		return result, fmt.Errorf("remove blocked: %d conflict(s)", result.Conflicts)
	}
	if opts.DryRun {
		return result, nil
	}
	for _, op := range result.Operations {
		if err := removeOperation(opts.Project, opts.Marker, op); err != nil {
			return result, err
		}
	}
	return result, nil
}

func FormatRemoveText(result *RemoveResult) string {
	groups := []struct {
		status string
		title  string
	}{
		{status: "removed", title: "Removed:"},
		{status: "updated", title: "Updated:"},
		{status: "missing", title: "Already missing:"},
		{status: "planned", title: "Dry-run planned:"},
		{status: "conflict", title: "Conflicts:"},
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Remove target: %s\n", result.Target)
	fmt.Fprintf(&b, "Project: %s\n", result.Project)
	if result.DryRun {
		b.WriteString("Mode: dry-run\n")
	}
	for _, group := range groups {
		var ops []RemoveOperation
		for _, op := range result.Operations {
			if op.Status == group.status {
				ops = append(ops, op)
			}
		}
		if len(ops) == 0 {
			continue
		}
		b.WriteString("\n")
		b.WriteString(group.title)
		b.WriteString("\n")
		for _, op := range ops {
			fmt.Fprintf(&b, "- %s", op.TargetPath)
			if op.Reason != "" {
				fmt.Fprintf(&b, " (%s)", op.Reason)
			}
			b.WriteString("\n")
		}
	}
	if result.Conflicts > 0 {
		fmt.Fprintf(&b, "\nRemove blocked: %d conflict(s)\n", result.Conflicts)
	}
	return b.String()
}

func FormatRemoveJSON(result *RemoveResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

func planTargetFile(loaded *pack.LoadedPack, targetProfile pack.TargetProfile, file pack.TargetProfileFile, opts PlanOptions) (PlanOperation, error) {
	generatedRel, err := generatedRelForPlan(targetProfile, file)
	if err != nil {
		return PlanOperation{}, err
	}
	sourcePath := filepath.Join(opts.From, filepath.FromSlash(generatedRel))
	targetRel, err := cleanPlanPath(file.TargetPath)
	if err != nil {
		return PlanOperation{}, err
	}
	targetPath := filepath.Join(opts.Project, filepath.FromSlash(targetRel))
	ownershipID := ownershipID(loaded.Manifest.Metadata.Name, targetRel)
	markerStyle := markerStyle(file)
	op := PlanOperation{
		TargetPath:    targetRel,
		GeneratedPath: filepath.ToSlash(sourcePath),
		OwnershipID:   ownershipID,
		MarkerStyle:   markerStyle,
	}
	generated, err := os.ReadFile(sourcePath)
	if err != nil {
		return PlanOperation{}, fmt.Errorf("read generated file %s: %w", sourcePath, err)
	}
	op.Preview = preview(generated)
	op.Content = string(generated)
	op.DisplayContent = string(generated)
	existing, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			if file.OwnedBlock {
				block := ownedBlock(opts.Marker, ownershipID, string(generated), markerStyle)
				op.Content = block
				op.DisplayContent = block
				op.Preview = preview([]byte(block))
				op.Diff = createFileDiff(targetRel, block)
			}
			op.Action = "create_file"
			op.Reason = "target path is missing"
			if op.Diff == "" {
				op.Diff = createFileDiff(targetRel, string(generated))
			}
			return op, nil
		}
		return PlanOperation{}, fmt.Errorf("read target file %s: %w", targetPath, err)
	}
	if file.OwnedBlock {
		if hasOwnedBlock(string(existing), opts.Marker, ownershipID, markerStyle) {
			if hasExactOwnedBlock(string(existing), opts.Marker, ownershipID, string(generated), markerStyle) {
				op.Action = "skip_existing_user_file"
				op.Reason = "target already matches generated content"
				return op, nil
			}
			op.Action = "update_owned_block"
			op.Reason = "Actlane ownership marker exists"
			op.DisplayContent = ownedBlock(opts.Marker, ownershipID, string(generated), markerStyle)
			op.Diff = ownedBlockDiff(targetRel, opts.Marker, ownershipID, string(generated), markerStyle)
			return op, nil
		}
		op.Action = "append_owned_block"
		op.Reason = "preserve existing user-owned content"
		op.DisplayContent = ownedBlock(opts.Marker, ownershipID, string(generated), markerStyle)
		op.Diff = appendBlockDiff(targetRel, opts.Marker, ownershipID, string(generated), markerStyle)
		return op, nil
	}
	if file.Owned {
		if string(existing) == string(generated) {
			op.Action = "skip_existing_user_file"
			op.Reason = "target already matches generated content"
			return op, nil
		}
		if hasOwnedBlock(string(existing), opts.Marker, ownershipID, markerStyle) {
			if hasExactOwnedBlock(string(existing), opts.Marker, ownershipID, string(generated), markerStyle) {
				op.Action = "skip_existing_user_file"
				op.Reason = "target already matches generated content"
				return op, nil
			}
			op.Action = "update_owned_block"
			op.Reason = "Actlane ownership marker exists"
			op.DisplayContent = ownedBlock(opts.Marker, ownershipID, string(generated), markerStyle)
			op.Diff = ownedBlockDiff(targetRel, opts.Marker, ownershipID, string(generated), markerStyle)
			return op, nil
		}
		if isActlaneOwnedJSONFile(targetRel, existing, loaded.Manifest.Metadata.Name) {
			op.Action = "update_owned_file"
			op.Reason = "target is an Actlane-owned JSON file"
			op.Diff = ownedBlockDiff(targetRel, opts.Marker, ownershipID, string(generated), markerStyle)
			return op, nil
		}
		op.Action = "conflict"
		op.Reason = "target path exists and is not Actlane-owned"
		return op, nil
	}
	op.Action = "conflict"
	op.Reason = "target profile file has no ownership strategy"
	return op, nil
}

func generatedRelForPlan(targetProfile pack.TargetProfile, file pack.TargetProfileFile) (string, error) {
	generated, err := cleanPlanPath(file.GeneratedPath)
	if err != nil {
		return "", err
	}
	root, err := cleanPlanPath(targetProfile.Spec.Output.Root)
	if err != nil {
		return "", err
	}
	if generated == root {
		return "", nil
	}
	prefix := root + "/"
	if strings.HasPrefix(generated, prefix) {
		return strings.TrimPrefix(generated, prefix), nil
	}
	return generated, nil
}

func targetProfileForPlan(loaded *pack.LoadedPack, target string) (pack.TargetProfile, error) {
	for _, targetProfile := range loaded.TargetProfiles {
		if targetProfile.Spec.Target == target {
			return targetProfile, nil
		}
	}
	return pack.TargetProfile{}, fmt.Errorf("unsupported target %q", target)
}

func targetFilesForAdoption(targetProfile pack.TargetProfile) ([]pack.TargetProfileFile, error) {
	switch targetProfile.Spec.Target {
	case "codex":
		return targetProfile.Spec.Codex.Files, nil
	case "opencode":
		return targetProfile.Spec.OpenCode.Files, nil
	default:
		return nil, fmt.Errorf("safe adoption currently supports codex and opencode only")
	}
}

func cleanPlanPath(filePath string) (string, error) {
	cleaned := path.Clean(filepath.ToSlash(strings.TrimSpace(filePath)))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("path is empty")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("path must be relative")
	}
	return cleaned, nil
}

func ownershipID(packName, targetPath string) string {
	return packName + "/" + targetPath
}

func hasOwnedBlock(content, marker, id, style string) bool {
	start, end := markerBounds(marker, id, style)
	return strings.Contains(content, start) && strings.Contains(content, end)
}

func hasExactOwnedBlock(content, marker, id, blockContent, style string) bool {
	return strings.Contains(content, ownedBlock(marker, id, blockContent, style))
}

func isActlaneOwnedJSONFile(targetPath string, content []byte, packName string) bool {
	if targetPath != "policies/policy-bundle.json" {
		return false
	}
	var value struct {
		Pack string `json:"pack"`
	}
	if err := json.Unmarshal(content, &value); err != nil {
		return false
	}
	return value.Pack == packName
}

func preview(content []byte) PlanPreview {
	sum := sha256.Sum256(content)
	return PlanPreview{
		Lines:  countLines(content),
		Bytes:  len(content),
		SHA256: fmt.Sprintf("%x", sum),
	}
}

func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	lines := strings.Count(string(content), "\n")
	if content[len(content)-1] != '\n' {
		lines++
	}
	return lines
}

func createFileDiff(targetPath, content string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "--- /dev/null\n+++ %s\n", targetPath)
	for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		fmt.Fprintf(&b, "+%s\n", line)
	}
	return b.String()
}

func appendBlockDiff(targetPath, marker, id, content, style string) string {
	return ownedBlockDiff(targetPath, marker, id, content, style)
}

func ownedBlockDiff(targetPath, marker, id, content, style string) string {
	block := ownedBlock(marker, id, content, style)
	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n+++ %s\n@@ actlane:%s\n", targetPath, targetPath, id)
	for _, line := range strings.Split(strings.TrimRight(block, "\n"), "\n") {
		fmt.Fprintf(&b, "+%s\n", line)
	}
	return b.String()
}

func ownedBlock(marker, id, content, style string) string {
	start, end := markerBounds(marker, id, style)
	return start + "\n" + strings.TrimRight(content, "\n") + "\n" + end + "\n"
}

func markerBounds(marker, id, style string) (string, string) {
	switch style {
	case "hash":
		return "# " + marker + ":start " + id, "# " + marker + ":end " + id
	default:
		return "<!-- " + marker + ":start " + id + " -->", "<!-- " + marker + ":end " + id + " -->"
	}
}

func markerStyle(file pack.TargetProfileFile) string {
	if file.MarkerStyle == "" {
		return "html"
	}
	return file.MarkerStyle
}

func applyStatus(action string, dryRun bool) string {
	if dryRun && action != "conflict" {
		return "planned"
	}
	switch action {
	case "create_file":
		return "created"
	case "append_owned_block":
		return "appended"
	case "update_owned_block":
		return "updated"
	case "update_owned_file":
		return "updated"
	case "skip_existing_user_file":
		return "skipped"
	case "conflict":
		return "conflict"
	default:
		return "skipped"
	}
}

func applyOperation(project, marker string, op PlanOperation) error {
	targetPath := filepath.Join(project, filepath.FromSlash(op.TargetPath))
	switch op.Action {
	case "create_file":
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		file, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return fmt.Errorf("create %s: %w", op.TargetPath, err)
		}
		defer file.Close()
		if _, err := file.WriteString(op.Content); err != nil {
			return fmt.Errorf("write %s: %w", op.TargetPath, err)
		}
	case "append_owned_block":
		existing, err := os.ReadFile(targetPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", op.TargetPath, err)
		}
		if hasOwnedBlock(string(existing), marker, op.OwnershipID, op.MarkerStyle) {
			return fmt.Errorf("append %s: Actlane ownership marker already exists", op.TargetPath)
		}
		next := string(existing)
		if next != "" && !strings.HasSuffix(next, "\n") {
			next += "\n"
		}
		if next != "" {
			next += "\n"
		}
		next += ownedBlock(marker, op.OwnershipID, op.Content, op.MarkerStyle)
		if err := os.WriteFile(targetPath, []byte(next), 0o644); err != nil {
			return fmt.Errorf("append %s: %w", op.TargetPath, err)
		}
	case "update_owned_block":
		existing, err := os.ReadFile(targetPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", op.TargetPath, err)
		}
		next, ok := replaceOwnedBlock(string(existing), marker, op.OwnershipID, op.Content, op.MarkerStyle)
		if !ok {
			return fmt.Errorf("update %s: Actlane ownership marker not found", op.TargetPath)
		}
		if err := os.WriteFile(targetPath, []byte(next), 0o644); err != nil {
			return fmt.Errorf("update %s: %w", op.TargetPath, err)
		}
	case "update_owned_file":
		if err := os.WriteFile(targetPath, []byte(op.Content), 0o644); err != nil {
			return fmt.Errorf("update %s: %w", op.TargetPath, err)
		}
	case "skip_existing_user_file":
		return nil
	default:
		return fmt.Errorf("unsupported apply action %q for %s", op.Action, op.TargetPath)
	}
	return nil
}

func replaceOwnedBlock(content, marker, id, nextContent, style string) (string, bool) {
	start, end := markerBounds(marker, id, style)
	startIndex := strings.Index(content, start)
	if startIndex < 0 {
		return "", false
	}
	endOffset := strings.Index(content[startIndex:], end)
	if endOffset < 0 {
		return "", false
	}
	endIndex := startIndex + endOffset + len(end)
	if endIndex < len(content) && content[endIndex] == '\n' {
		endIndex++
	}
	return content[:startIndex] + ownedBlock(marker, id, nextContent, style) + content[endIndex:], true
}

func planRemoveOperation(project, marker, sourcePath, targetRel, ownershipID string, file pack.TargetProfileFile) RemoveOperation {
	op := RemoveOperation{
		Action:      removeAction(file),
		TargetPath:  targetRel,
		OwnershipID: ownershipID,
		MarkerStyle: markerStyle(file),
	}
	targetPath := filepath.Join(project, filepath.FromSlash(targetRel))
	existing, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			op.Status = "missing"
			op.Reason = "target path is missing"
			return op
		}
		op.Status = "conflict"
		op.Reason = err.Error()
		return op
	}
	if file.OwnedBlock {
		if hasOwnedBlock(string(existing), marker, ownershipID, op.MarkerStyle) {
			op.Status = "updated"
			op.Reason = "remove Actlane ownership block"
			return op
		}
		op.Status = "missing"
		op.Reason = "Actlane ownership marker not found"
		return op
	}
	if file.Owned {
		generated, err := os.ReadFile(sourcePath)
		if err != nil {
			op.Status = "conflict"
			if os.IsNotExist(err) {
				op.Reason = "generated source is missing; run generate or provide --from"
			} else {
				op.Reason = "generated source is unreadable: " + err.Error()
			}
			return op
		}
		if string(existing) == string(generated) {
			op.Status = "removed"
			op.Reason = "target matches generated content"
			return op
		}
		op.Status = "conflict"
		op.Reason = "target exists but does not match generated content"
		return op
	}
	op.Status = "conflict"
	op.Reason = "target profile file has no ownership strategy"
	return op
}

func removeAction(file pack.TargetProfileFile) string {
	if file.OwnedBlock {
		return "remove_owned_block"
	}
	return "remove_file"
}

func removeOperation(project, marker string, op RemoveOperation) error {
	targetPath := filepath.Join(project, filepath.FromSlash(op.TargetPath))
	switch op.Action {
	case "remove_file":
		if op.Status == "missing" {
			return nil
		}
		if op.Status != "removed" {
			return fmt.Errorf("remove %s: unsafe status %s", op.TargetPath, op.Status)
		}
		if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", op.TargetPath, err)
		}
	case "remove_owned_block":
		if op.Status == "missing" {
			return nil
		}
		if op.Status != "updated" {
			return fmt.Errorf("remove block %s: unsafe status %s", op.TargetPath, op.Status)
		}
		existing, err := os.ReadFile(targetPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", op.TargetPath, err)
		}
		next, ok := deleteOwnedBlock(string(existing), marker, op.OwnershipID, op.MarkerStyle)
		if !ok {
			return fmt.Errorf("remove block %s: Actlane ownership marker not found", op.TargetPath)
		}
		if strings.TrimSpace(next) == "" {
			if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove %s: %w", op.TargetPath, err)
			}
			return nil
		}
		if err := os.WriteFile(targetPath, []byte(next), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", op.TargetPath, err)
		}
	default:
		return fmt.Errorf("unsupported remove action %q for %s", op.Action, op.TargetPath)
	}
	return nil
}

func deleteOwnedBlock(content, marker, id, style string) (string, bool) {
	start, end := markerBounds(marker, id, style)
	startIndex := strings.Index(content, start)
	if startIndex < 0 {
		return "", false
	}
	endOffset := strings.Index(content[startIndex:], end)
	if endOffset < 0 {
		return "", false
	}
	endIndex := startIndex + endOffset + len(end)
	if endIndex < len(content) && content[endIndex] == '\n' {
		endIndex++
	}
	if startIndex > 0 && content[startIndex-1] == '\n' {
		startIndex--
	}
	return content[:startIndex] + content[endIndex:], true
}

func formatFenced(content, lang string, numbered bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "```%s\n", lang)
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	width := len(fmt.Sprintf("%d", len(lines)))
	for i, line := range lines {
		if numbered {
			fmt.Fprintf(&b, "%*d | %s\n", width, i+1, line)
		} else {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("```\n")
	return b.String()
}
