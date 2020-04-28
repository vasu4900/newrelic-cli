release_script ?= ./scripts/release.sh

gotools += github.com/goreleaser/goreleaser

rel_cmd ?= goreleaser
dist_dir ?= ./dist

homebrew_cmd ?= brew
homebrew_upstream ?= git@github.com:newrelic-forks/homebrew-core.git
archive_url       ?= https://github.com/newrelic/$(strip $(project_name))/archive/v$(strip $(project_ver_tagged)).tar.gz

# example usage: make release version=0.11.0
release: build
	@echo "=== $(project_name) === [ release          ]: generating release."
	$(release_script) $(version)

release-clean:
	@echo "=== $(project_name) === [ release-clean    ]: distribution files..."
	@rm -rfv $(dist_dir) $(srcdir)/tmp

release-publish: clean tools docker-login snapcraft-login release-notes
	@echo "=== $(project_name) === [ release-publish  ]: publishing release via $(rel_cmd)"
	$(rel_cmd) --release-notes=$(srcdir)/tmp/$(release_notes_file)

# local snapshot
snapshot: clean tools release-notes
	@echo "=== $(project_name) === [ snapshot         ]: creating release via $(rel_cmd)"
	@echo "=== $(project_name) === [ snapshot         ]:   this will not be published!"
	$(rel_cmd) --skip-publish --snapshot --release-notes=$(srcdir)/tmp/$(release_notes_file)

release-homebrew:
ifeq ($(homebrew_github_api_token), "")
	@echo "=== $(project_name) === [ admin-homebrew   ]: homebrew_github_api_token must be set"
	exit 1
endif
ifeq ($(shell which $(homebrew_cmd)), "")
	@echo "=== $(project_name) === [ admin-homebrew   ]: hombrew command '$(homebrew_cmd)' not found."
	exit 1
endif
	@echo "=== $(project_name) === [ admin-homebrew   ]: updating homebrew..."
	@hub_remote=$(homebrew_upstream) $(homebrew_cmd) bump-formula-pr --url $(archive_url) $(project_name)

.phony: release release-clean release-homebrew release-publish snapshot
