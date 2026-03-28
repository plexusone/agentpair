// Package main provides the CLI entry point for agentpair.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/plexusone/agentpair/internal/agent"
	"github.com/plexusone/agentpair/internal/agent/claude"
	"github.com/plexusone/agentpair/internal/agent/codex"
	"github.com/plexusone/agentpair/internal/config"
	"github.com/plexusone/agentpair/internal/dashboard"
	"github.com/plexusone/agentpair/internal/logger"
	"github.com/plexusone/agentpair/internal/loop"
	"github.com/plexusone/agentpair/internal/run"
	"github.com/plexusone/agentpair/internal/tmux"
	"github.com/plexusone/agentpair/internal/update"
	"github.com/plexusone/agentpair/internal/worktree"
)

var (
	// Version is set at build time.
	version = "dev"

	// Flags
	cfg = config.DefaultConfig()

	// Root command
	rootCmd = &cobra.Command{
		Use:   "agentpair",
		Short: "Agent-to-agent pair programming between Claude and Codex",
		Long: `AgentPair orchestrates pair programming sessions between AI agents.
One agent works on the task while the other reviews, iterating until completion.`,
		PersistentPreRunE: loadConfigFile,
		RunE:              runMain,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}
)

func init() {
	// Main flags
	rootCmd.Flags().StringVarP(&cfg.Prompt, "prompt", "p", "", "Task prompt")
	rootCmd.Flags().StringVarP(&cfg.Agent, "agent", "a", "codex", "Primary worker agent (claude or codex)")
	rootCmd.Flags().IntVarP(&cfg.MaxIterations, "max-iterations", "m", 20, "Maximum loop iterations")
	rootCmd.Flags().StringVar(&cfg.Proof, "proof", "", "Proof/verification command")
	rootCmd.Flags().StringVar(&cfg.ReviewMode, "review", "claudex", "Review mode: claude, codex, or claudex")
	rootCmd.Flags().StringVar(&cfg.DoneSignal, "done", "DONE", "Custom done signal")

	// Mode flags
	rootCmd.Flags().BoolVar(&cfg.ClaudeOnly, "claude-only", false, "Run Claude in single-agent mode")
	rootCmd.Flags().BoolVar(&cfg.CodexOnly, "codex-only", false, "Run Codex in single-agent mode")

	// Workspace flags
	rootCmd.Flags().BoolVar(&cfg.UseTmux, "tmux", false, "Use tmux for side-by-side panes")
	rootCmd.Flags().BoolVar(&cfg.UseWorktree, "worktree", false, "Create git worktree for isolation")

	// Resume flags
	rootCmd.Flags().IntVar(&cfg.RunID, "run-id", 0, "Resume by run ID")
	rootCmd.Flags().StringVar(&cfg.SessionID, "session", "", "Resume by session ID")

	// Other flags
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Verbose output")

	// Subcommands
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(bridgeCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(versionCmd)
}

// loadConfigFile loads the config file and applies it to cfg.
// CLI flags take precedence over config file values.
func loadConfigFile(cmd *cobra.Command, args []string) error {
	fileCfg, err := config.LoadConfigFile()
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	if fileCfg == nil {
		return nil // No config file
	}

	// Apply file config values only for flags that weren't explicitly set
	flags := cmd.Flags()

	if !flags.Changed("agent") && fileCfg.Agent != "" {
		cfg.Agent = fileCfg.Agent
	}
	if !flags.Changed("max-iterations") && fileCfg.MaxIterations > 0 {
		cfg.MaxIterations = fileCfg.MaxIterations
	}
	if !flags.Changed("proof") && fileCfg.Proof != "" {
		cfg.Proof = fileCfg.Proof
	}
	if !flags.Changed("review") && fileCfg.ReviewMode != "" {
		cfg.ReviewMode = fileCfg.ReviewMode
	}
	if !flags.Changed("done") && fileCfg.DoneSignal != "" {
		cfg.DoneSignal = fileCfg.DoneSignal
	}
	if !flags.Changed("tmux") && fileCfg.UseTmux {
		cfg.UseTmux = fileCfg.UseTmux
	}
	if !flags.Changed("worktree") && fileCfg.UseWorktree {
		cfg.UseWorktree = fileCfg.UseWorktree
	}
	if !flags.Changed("verbose") && fileCfg.Verbose {
		cfg.Verbose = fileCfg.Verbose
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMain(cmd *cobra.Command, args []string) error {
	// Setup context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Initialize logger
	logLevel := slog.LevelInfo
	if cfg.Verbose {
		logLevel = slog.LevelDebug
	}
	log := logger.New(&logger.Options{Level: logLevel})
	ctx = logger.NewContext(ctx, log)

	log.Info("starting agentpair", "version", version)

	// Check for updates in background
	updater := update.New(version)
	go updater.PrintUpdateNotice(ctx)

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	cfg.RepoPath = workDir
	log.Debug("working directory", "path", workDir)

	// Setup paths
	paths, err := config.NewPaths()
	if err != nil {
		return fmt.Errorf("failed to setup paths: %w", err)
	}

	// Get prompt from args if not specified
	if cfg.Prompt == "" && len(args) > 0 {
		cfg.Prompt = args[0]
	}

	// Handle resume
	if cfg.RunID > 0 || cfg.SessionID != "" {
		log.Info("resuming run", "run_id", cfg.RunID, "session_id", cfg.SessionID)
		return resumeRun(ctx, paths)
	}

	// Validate prompt
	if cfg.Prompt == "" {
		return fmt.Errorf("prompt is required (use --prompt or provide as argument)")
	}

	// Validate mode flags
	if cfg.ClaudeOnly && cfg.CodexOnly {
		return fmt.Errorf("cannot specify both --claude-only and --codex-only")
	}

	// Create run manager
	manager := run.NewManager(paths, cfg.RepoPath)

	// Handle worktree
	var wt *worktree.Worktree
	if cfg.UseWorktree {
		if !worktree.IsGitRepo(cfg.RepoPath) {
			return fmt.Errorf("--worktree requires a git repository")
		}

		// Will be set after run is created
	}

	// Create new run
	r, err := manager.Create(cfg.Prompt, cfg)
	if err != nil {
		return fmt.Errorf("failed to create run: %w", err)
	}
	defer r.Close()

	log = logger.WithRunID(log, r.Manifest.ID)
	ctx = logger.NewContext(ctx, log)
	log.Info("created run",
		"prompt", cfg.Prompt,
		"agent", cfg.PrimaryAgent(),
		"review_mode", cfg.ReviewMode,
		"max_iterations", cfg.MaxIterations)

	// Setup worktree if enabled
	if cfg.UseWorktree {
		wtPath := worktree.GenerateWorktreePath(cfg.RepoPath, r.Manifest.ID)
		branch := worktree.GenerateBranchName(r.Manifest.ID)
		wt = worktree.New(cfg.RepoPath, wtPath, branch)

		log.Info("creating worktree", "path", wtPath, "branch", branch)
		if err := wt.Create(ctx); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
		defer func() {
			if wt.WasCreated() {
				log.Debug("removing worktree", "path", wtPath)
				wt.Remove(context.Background(), false)
			}
		}()

		// Update working directory
		cfg.RepoPath = wt.Path()
		r.Manifest.WorktreePath = wt.Path()
		r.Save()
	}

	// Handle tmux
	var tmuxSession *tmux.Session
	if cfg.UseTmux {
		if !tmux.IsTmuxAvailable() {
			return fmt.Errorf("--tmux requires tmux to be installed")
		}

		repoName := filepath.Base(cfg.RepoPath)
		sessionName := tmux.GenerateSessionName(repoName, r.Manifest.ID)
		tmuxSession = tmux.NewSession(sessionName, cfg.RepoPath)

		log.Info("creating tmux session", "name", sessionName)
		if err := tmuxSession.Create(ctx); err != nil {
			return fmt.Errorf("failed to create tmux session: %w", err)
		}

		layout := tmux.NewLayout(tmuxSession)
		if err := layout.PairedLayout(ctx); err != nil {
			return fmt.Errorf("failed to setup tmux layout: %w", err)
		}

		r.Manifest.TmuxSession = sessionName
		r.Save()

		fmt.Printf("Created tmux session: %s\n", sessionName)
		fmt.Printf("Run 'tmux attach -t %s' to view\n", sessionName)
	}

	// Create agents
	agentCfg := &agent.Config{
		WorkDir:     cfg.RepoPath,
		Prompt:      cfg.Prompt,
		AutoApprove: true,
		Verbose:     cfg.Verbose,
	}

	// Run appropriate loop
	if cfg.IsSingleAgentMode() {
		log.Info("starting single-agent mode", "agent", cfg.PrimaryAgent())
		return runSingleAgent(ctx, r, agentCfg)
	}

	log.Info("starting paired-agent mode", "primary", cfg.Agent, "secondary", cfg.SecondaryAgent())
	return runPairedAgents(ctx, r, agentCfg)
}

func runSingleAgent(ctx context.Context, r *run.Run, agentCfg *agent.Config) error {
	var a agent.Agent

	if cfg.ClaudeOnly {
		a = claude.New(agentCfg)
	} else {
		a = codex.New(agentCfg)
	}

	l := loop.NewSingle(cfg, r, a)
	return l.Run(ctx)
}

func runPairedAgents(ctx context.Context, r *run.Run, agentCfg *agent.Config) error {
	var primary, secondary agent.Agent

	if cfg.Agent == "claude" {
		primary = claude.New(agentCfg)
		secondary = codex.New(agentCfg)
	} else {
		primary = codex.New(agentCfg)
		secondary = claude.New(agentCfg)
	}

	l := loop.New(cfg, r, primary, secondary)
	return l.Run(ctx)
}

func resumeRun(ctx context.Context, paths *config.Paths) error {
	manager := run.NewManager(paths, cfg.RepoPath)

	var r *run.Run
	var err error

	if cfg.RunID > 0 {
		r, err = manager.Load(cfg.RunID)
	} else {
		r, err = manager.FindBySessionID(cfg.SessionID)
	}

	if err != nil {
		return fmt.Errorf("failed to load run: %w", err)
	}
	defer r.Close()

	// Restore config from manifest
	cfg.Prompt = r.Manifest.Prompt
	cfg.MaxIterations = r.Manifest.MaxIterations
	cfg.ReviewMode = r.Manifest.ReviewMode
	cfg.DoneSignal = r.Manifest.DoneSignal
	cfg.Agent = r.Manifest.PrimaryAgent

	// Determine if single-agent mode
	cfg.ClaudeOnly = r.Manifest.ReviewMode == "claude" && r.Manifest.CodexSessionID == ""
	cfg.CodexOnly = r.Manifest.ReviewMode == "codex" && r.Manifest.ClaudeSessionID == ""

	agentCfg := &agent.Config{
		WorkDir:     r.Manifest.RepoPath,
		Prompt:      cfg.Prompt,
		AutoApprove: true,
		Verbose:     cfg.Verbose,
	}

	// Resume with session IDs
	if r.Manifest.ClaudeSessionID != "" {
		agentCfg.SessionID = r.Manifest.ClaudeSessionID
	}

	if cfg.IsSingleAgentMode() {
		var a agent.Agent
		if cfg.ClaudeOnly {
			a = claude.New(agentCfg)
		} else {
			a = codex.New(agentCfg)
		}
		l := loop.NewSingle(cfg, r, a)
		return l.Resume(ctx)
	}

	var primary, secondary agent.Agent
	claudeCfg := &agent.Config{
		WorkDir:     r.Manifest.RepoPath,
		SessionID:   r.Manifest.ClaudeSessionID,
		AutoApprove: true,
	}
	codexCfg := &agent.Config{
		WorkDir:     r.Manifest.RepoPath,
		SessionID:   r.Manifest.CodexSessionID,
		AutoApprove: true,
	}

	if cfg.Agent == "claude" {
		primary = claude.New(claudeCfg)
		secondary = codex.New(codexCfg)
	} else {
		primary = codex.New(codexCfg)
		secondary = claude.New(claudeCfg)
	}

	l := loop.New(cfg, r, primary, secondary)
	return l.Resume(ctx)
}

// Dashboard command
var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show live dashboard of active runs",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		paths, err := config.NewPaths()
		if err != nil {
			return err
		}

		d := dashboard.New(paths)
		return d.Run(ctx)
	},
}

// Bridge command
var bridgeCmd = &cobra.Command{
	Use:   "bridge",
	Short: "Show bridge status for a run",
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, err := config.NewPaths()
		if err != nil {
			return err
		}

		workDir, _ := os.Getwd()
		manager := run.NewManager(paths, workDir)

		if cfg.RunID == 0 {
			// Show status of all runs
			ids, err := manager.List()
			if err != nil {
				return err
			}

			for _, id := range ids {
				r, err := manager.Load(id)
				if err != nil {
					continue
				}
				status := r.Bridge.Status()
				fmt.Printf("Run #%d: %s\n", id, status.String())
				r.Close()
			}
			return nil
		}

		// Show specific run
		r, err := manager.Load(cfg.RunID)
		if err != nil {
			return err
		}
		defer r.Close()

		status := r.Bridge.Status()
		fmt.Printf("Run #%d Bridge Status:\n", cfg.RunID)
		fmt.Printf("  Total Messages: %d\n", status.TotalMessages)
		fmt.Printf("  Done Signal: %v\n", status.HasDoneSignal)
		fmt.Printf("  Pass Count: %d\n", status.PassCount)
		fmt.Printf("  Fail Count: %d\n", status.FailCount)
		fmt.Println("  By Agent:")
		for agent, count := range status.ByAgent {
			fmt.Printf("    %s: %d\n", agent, count)
		}
		return nil
	},
}

// Update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install updates",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		updater := update.New(version)

		fmt.Println("Checking for updates...")
		result, err := updater.Check(ctx)
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}

		if !result.UpdateAvailable {
			fmt.Printf("You're running the latest version (%s)\n", result.CurrentVersion)
			return nil
		}

		fmt.Printf("Update available: %s → %s\n", result.CurrentVersion, result.LatestVersion)
		fmt.Println("Installing update...")

		if err := updater.Update(ctx, result.DownloadURL); err != nil {
			return fmt.Errorf("failed to install update: %w", err)
		}

		fmt.Println("Update installed successfully!")
		fmt.Println("Please restart agentpair to use the new version.")
		return nil
	},
}

// Version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("agentpair version %s\n", version)
	},
}
