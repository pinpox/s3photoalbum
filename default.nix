{ lib, pkgs, config, inputs, ... }:
with lib;

let cfg = config.services.s3photoalbum;
in {

  options.services.s3photoalbum = {

    enable = mkEnableOption "s3photoalbum";

    dataDir = mkOption {
      type = types.str;
      default = "/var/lib/s3photoalbum";
      description = ''
        The directory where s3photoalbum stores its data files.
      '';
    };

    envfile = mkOption {
      type = types.str;
      default = "/var/src/secrets/s3photoalbum/envfile";
      description = ''
        The location of the envfile containing secrets
      '';
    };

    openFirewall = mkOption {
      type = types.bool;
      default = false;
      description = ''
        Open the appropriate ports in the firewall for s3photoalbum.
      '';
    };

    user = mkOption {
      type = types.str;
      default = "s3photoalbum";
      description = "User account under which s3photoalbum runs.";
    };

    group = mkOption {
      type = types.str;
      default = "s3photoalbum";
      description = "Group under which s3photoalbum runs.";
    };

  };

  config = mkIf cfg.enable {

    systemd.services.s3photoalbum = {
      description = "A self-hosted photo album";
      wantedBy = [ "multi-user.target" ];

      serviceConfig = mkMerge [
        {
          EnvironmentFile = [ cfg.envfile ];
          User = cfg.user;
          Group = cfg.group;
          WorkingDirectory = cfg.dataDir;
          ExecStart = "${pkgs.s3photoalbum}/bin/s3photoalbum";
          Restart = "on-failure";
        }
        (mkIf (cfg.dataDir == "/var/lib/s3photoalbum") {
          StateDirectory = "s3photoalbum";
        })
      ];
    };

    users.users = mkIf (cfg.user == "s3photoalbum") {
      s3photoalbum = {
        isSystemUser = true;
        group = cfg.group;
        description = "s3photoalbum system user";
      };
    };

    users.groups = mkIf (cfg.group == "s3photoalbum") { s3photoalbum = { }; };

    networking.firewall = mkIf cfg.openFirewall { allowedTCPPorts = [ 8083 ]; };

  };
  meta = { maintainers = with lib.maintainers; [ mayniklas ]; };
}
