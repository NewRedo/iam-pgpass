{
  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
  };
  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        name = "iam-pgpass";
        version = "0.0.1";
      in rec {
        packages = {
          default = pkgs.buildGoModule rec {
            pname = name;
            inherit version;
            src = ./.;

            vendorHash = "sha256-fdcnTGLHJO9j90yW2AJ4BZmu4TC+u8uaaNW4d95LpGQ=";

            meta = {
              description =
                "A simple tool to manage IAM credentials for AWS RDS instances";
              license = "GPLv3";
            };
          };
          docker = pkgs.dockerTools.buildImage {
            inherit name;
            tag = version;

            copyToRoot = [ packages.default pkgs.dockerTools.binSh ];

            config = { Cmd = [ "/bin/iam-pgpass" ]; };
          };
        };

        devShell = pkgs.mkShell { buildInputs = with pkgs; [ go ]; };
      });
}
