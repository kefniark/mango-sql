{ pkgs, lib, config, inputs, ... }:

let
  pkgs-unstable = import inputs.nixpkgs-unstable { system = pkgs.stdenv.system; };
in {
  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    
    # Golang
    pkgs-unstable.go_1_23
    pkgs-unstable.gotools
    pkgs-unstable.golangci-lint
    pkgs-unstable.gocover-cobertura

    # NodeJS (docs)
    pkgs.nodejs_20

    # CLI Tools
    pkgs.goose
    pkgs.just
    pkgs.ko
    pkgs.upx
  ];

  # https://devenv.sh/tests/
  enterTest = ''
  just generate
  just test
  '';

  # https://devenv.sh/services/
  # services.postgres.enable = true;

  # https://devenv.sh/languages/
  # languages.nix.enable = true;

  # https://devenv.sh/pre-commit-hooks/
  # pre-commit.hooks.shellcheck.enable = true;

  # https://devenv.sh/processes/
  # processes.ping.exec = "ping example.com";

  processes.postgres = {
    exec = ( lib.concatStringsSep " " [
        "docker compose up -d && sleep 5"
      ]
    );

    process-compose = {
      description = "docker container of postgres";
      shutdown = {
        command = "docker compose down";
      };
    };
  };

  # See full reference at https://devenv.sh/reference/options/
}
