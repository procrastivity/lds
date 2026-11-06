{
  description = "lds — dev shell and package for the lds Go CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        # A flake sees rev/shortRev/dirtyRev and lastModified — never tags.
        # So the Nix path stamps the commit where the make path stamps the
        # release tag (C1.7): `nix build` reports `abc1234`, `make build`
        # at a tag reports `v0.1.0`. Both name the same commit, and for
        # a remote build at a tag resolves the rev to that tag's commit —
        # a stricter identifier, not a looser one. Do not "fix"
        # this with a VERSION file: the tag is the single source of truth,
        # and a second copy would go stale in silence.
        version = self.shortRev or self.dirtyShortRev or "dev";

        lds = pkgs.buildGoModule {
          pname = "lds";
          inherit version;
          src = ./.;
          # Dependencies are vendored (vendor/ is committed), so there is
          # no fixed-output fetch and no hash to re-pin when go.mod
          # changes — run `go mod vendor` after a dependency change and
          # commit the result instead.
          vendorHash = null;

          env.CGO_ENABLED = 0;

          ldflags = [
            "-X main.version=${version}"
            "-X main.commit=${self.rev or self.dirtyRev or "unknown"}"
            # A fixed epoch, not an oversight: a real build date would make
            # the derivation unreproducible. `commit` above carries the
            # provenance the date would otherwise supply.
            "-X main.date=1970-01-01T00:00:00Z"
          ];

          subPackages = [ "cmd/lds" ];

          postInstall = ''
            mkdir -p $out/share/lds
            cp -r assets $out/share/lds/assets
            # assets.go only exists so `assets/` can embed itself as the
            # binary's last-resort fallback; it is source, not a shipped
            # asset, and must not appear in the installed share tree (C1.6).
            rm -f $out/share/lds/assets/assets.go
          '';

          meta = {
            description = "lds — Layered Documentation System CLI (scaffold)";
            license = pkgs.lib.licenses.mit;
            mainProgram = "lds";
          };
        };
      in {
        packages.default = lds;

        devShells.default = pkgs.mkShell {
          name = "lds";
          packages = with pkgs; [
            go
            golangci-lint
            gofumpt
            git-cliff
            gnumake
            pre-commit
            shellcheck
            # Execute the documented offline tutorial in the CLI tests.
            bash
            coreutils
            python3
          ];

          # stderr, not stdout: `nix develop --command lds manifest
          # --json | jq` has to work, and anything this hook prints to
          # stdout lands in front of the document (C2.1 in spirit — the
          # tool owns its stdout, and so must its environment).
          shellHook = ''
            echo "lds dev shell — run 'make check' to lint+test, 'make hooks' to install pre-commit." >&2
          '';
        };
      }) // {
      # Downstream flakes consume `pkgs.lds` through this overlay;
      # keep it or their inputs break.
      overlays.default = final: prev: {
        "lds" = self.packages.${prev.system}.default;
      };
    };
}
