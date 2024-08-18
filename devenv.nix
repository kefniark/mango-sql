{ pkgs, lib, config, inputs, ... }:

{
  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    
    # Golang
    pkgs.go_1_22
    pkgs.gotools
    pkgs.golangci-lint
    pkgs.gocover-cobertura

    # NodeJS (docs)
    pkgs.nodejs_20

    # CLI Tools
    pkgs.goose
    pkgs.just
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
