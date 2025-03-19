{
  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
  };
  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = import nixpkgs { inherit system; };
      in {
        defaultPackage = pkgs.buildGoModule rec {
          pname = "iam-pgpass";
          version = "0.0.1";
          src = ./.;

          vendorHash = "sha256-fdcnTGLHJO9j90yW2AJ4BZmu4TC+u8uaaNW4d95LpGQ=";

          meta = {
            description = "A simple tool to manage IAM credentials for AWS RDS instances";
            license = "GPLv3";
          };
        };
        devShell = pkgs.mkShell { buildInputs = with pkgs; [ go ]; };
      });
}
