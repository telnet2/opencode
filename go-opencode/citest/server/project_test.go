package server_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencode-ai/opencode/citest/testutil"
	"github.com/opencode-ai/opencode/pkg/types"
)

var _ = Describe("Project Endpoints", func() {
	Describe("GET /project", func() {
		It("should return a list of projects", func() {
			resp, err := client.Get(ctx, "/project")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())
			Expect(resp.Headers.Get("Content-Type")).To(ContainSubstring("application/json"))

			var projects []types.Project
			err = resp.JSON(&projects)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(projects)).To(BeNumerically(">=", 1), "Should return at least one project")
		})

		It("should return projects with required fields", func() {
			resp, err := client.Get(ctx, "/project")
			Expect(err).NotTo(HaveOccurred())

			var projects []types.Project
			err = resp.JSON(&projects)
			Expect(err).NotTo(HaveOccurred())

			for _, p := range projects {
				Expect(p.ID).NotTo(BeEmpty(), "Project ID should not be empty")
				Expect(p.Worktree).NotTo(BeEmpty(), "Project worktree should not be empty")
				Expect(p.Time.Created).To(BeNumerically(">", 0), "Project created time should be positive")
			}
		})
	})

	Describe("GET /project/current", func() {
		It("should return the current project", func() {
			resp, err := client.Get(ctx, "/project/current")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.IsSuccess()).To(BeTrue())
			Expect(resp.Headers.Get("Content-Type")).To(ContainSubstring("application/json"))

			var project types.Project
			err = resp.JSON(&project)
			Expect(err).NotTo(HaveOccurred())

			Expect(project.ID).NotTo(BeEmpty())
			Expect(project.Worktree).NotTo(BeEmpty())
		})

		It("should return a project with valid worktree path", func() {
			resp, err := client.Get(ctx, "/project/current")
			Expect(err).NotTo(HaveOccurred())

			var project types.Project
			err = resp.JSON(&project)
			Expect(err).NotTo(HaveOccurred())

			// Worktree path should be absolute
			Expect(filepath.IsAbs(project.Worktree)).To(BeTrue(), "Worktree should be an absolute path")

			// Worktree directory should exist (or be the temp dir used by test server)
			_, err = os.Stat(project.Worktree)
			Expect(err).NotTo(HaveOccurred(), "Worktree directory should exist")
		})

		It("should return project with unique ID", func() {
			resp, err := client.Get(ctx, "/project/current")
			Expect(err).NotTo(HaveOccurred())

			var project types.Project
			err = resp.JSON(&project)
			Expect(err).NotTo(HaveOccurred())

			// ID should be a 16-character hex string (hash prefix)
			Expect(len(project.ID)).To(Equal(16), "Project ID should be 16 characters")
			for _, c := range project.ID {
				Expect(c >= '0' && c <= '9' || c >= 'a' && c <= 'f').To(BeTrue(),
					"Project ID should be hexadecimal")
			}
		})

		It("should return consistent ID for same directory", func() {
			resp1, err := client.Get(ctx, "/project/current")
			Expect(err).NotTo(HaveOccurred())

			var project1 types.Project
			err = resp1.JSON(&project1)
			Expect(err).NotTo(HaveOccurred())

			resp2, err := client.Get(ctx, "/project/current")
			Expect(err).NotTo(HaveOccurred())

			var project2 types.Project
			err = resp2.JSON(&project2)
			Expect(err).NotTo(HaveOccurred())

			Expect(project1.ID).To(Equal(project2.ID), "Same directory should return same project ID")
		})

		It("should detect VCS when present", func() {
			// Create a temp directory with .git folder
			tempDir, err := testutil.NewTempDir()
			Expect(err).NotTo(HaveOccurred())
			defer tempDir.Cleanup()

			// Create a .git directory to simulate a git repo
			gitDir := filepath.Join(tempDir.Path, ".git")
			err = os.MkdirAll(gitDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			// Request with directory parameter
			resp, err := client.Get(ctx, "/project/current", testutil.WithQuery(map[string]string{
				"directory": tempDir.Path,
			}))
			Expect(err).NotTo(HaveOccurred())

			var project types.Project
			err = resp.JSON(&project)
			Expect(err).NotTo(HaveOccurred())

			Expect(project.VCS).To(Equal("git"), "Should detect git VCS")
		})

		It("should not set VCS when .git is absent", func() {
			// Create a temp directory without .git folder
			tempDir, err := testutil.NewTempDir()
			Expect(err).NotTo(HaveOccurred())
			defer tempDir.Cleanup()

			// Request with directory parameter
			resp, err := client.Get(ctx, "/project/current", testutil.WithQuery(map[string]string{
				"directory": tempDir.Path,
			}))
			Expect(err).NotTo(HaveOccurred())

			var project types.Project
			err = resp.JSON(&project)
			Expect(err).NotTo(HaveOccurred())

			Expect(project.VCS).To(BeEmpty(), "Should not set VCS when .git is absent")
		})
	})

	Describe("Project list consistency", func() {
		It("should return current project in list", func() {
			// Get current project
			currentResp, err := client.Get(ctx, "/project/current")
			Expect(err).NotTo(HaveOccurred())

			var currentProject types.Project
			err = currentResp.JSON(&currentProject)
			Expect(err).NotTo(HaveOccurred())

			// Get project list
			listResp, err := client.Get(ctx, "/project")
			Expect(err).NotTo(HaveOccurred())

			var projects []types.Project
			err = listResp.JSON(&projects)
			Expect(err).NotTo(HaveOccurred())

			// Current project should be in the list
			found := false
			for _, p := range projects {
				if p.ID == currentProject.ID {
					found = true
					Expect(p.Worktree).To(Equal(currentProject.Worktree))
					break
				}
			}
			Expect(found).To(BeTrue(), "Current project should be in the list")
		})
	})
})
