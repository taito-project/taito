{
  description = "Development environment for taito";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            just
            nfpm
            gettext
          ];

          shellHook = ''
            echo "Taito development environment loaded!"
            echo "Run 'just' to see available commands."
          '';
        };
      }
    );
}
