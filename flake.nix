{
  description = "Go 1.23 workspace";

  inputs.nixpkgs.url = "nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    # Add dependencies that are only needed for development
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default =
          let
            p = pkgs;
          in
          pkgs.mkShell {
            buildInputs = [
              p.go
              p.gopls
              p.gotools
              p.go-tools
            ];
          };
      }
    );
}
