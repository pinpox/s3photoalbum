{ lib, pkgs, config, ... }:
with lib;

let cfg = config.services.s3photoalbum;
in {

  options.services.s3photoalbum = {

    enable = mkEnableOption "s3photoalbum";

    acme = mkOption {
      type = types.bool;
      default = false;
      description = ''
        Configure nginx to use ACME
      '';
    };

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

    hostname = mkOption {
      type = types.str;
      default = "gallery.your-domain.com";
      description = ''
        hostname to use for nginx
      '';
    };

    nginx = mkOption {
      type = types.bool;
      default = false;
      description = ''
        configure nginx s3photoalbum virtualHost 
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

    services.nginx = mkIf cfg.nginx {
      enable = true;
      recommendedProxySettings = true;
      recommendedTlsSettings = mkIf cfg.acme true;
      virtualHosts = {
        "${cfg.hostname}" = {
          forceSSL = mkIf cfg.acme true;
          enableACME = mkIf cfg.acme true;
          locations."/" = { proxyPass = "http://127.0.0.1:7788"; };
        };
      };
    };

    systemd.services.s3photoalbum = {
      description = "A self-hosted photo album";
      wantedBy = [ "multi-user.target" ];

      serviceConfig = mkMerge [
        {
          EnvironmentFile = [ cfg.envfile ];
          User = cfg.user;
          Group = cfg.group;
          WorkingDirectory = cfg.dataDir;
          ExecStart = "${pkgs.s3photoalbum}/bin/server";
          Restart = "on-failure";
          Environment = [ "RESOURCES_DIR='${pkgs.s3photoalbum}/share'" ];
        }
        (mkIf (cfg.dataDir == "/var/lib/s3photoalbum") {
          StateDirectory = "s3photoalbum";
        })
      ];
    };

    systemd.services.s3photoalbum-thumbnailer = {
      description = "A self-hosted photo album - thumbnailer service";
      wantedBy = [ "multi-user.target" ];

      serviceConfig = mkMerge [
        {
          EnvironmentFile = [ cfg.envfile ];
          User = cfg.user;
          Group = cfg.group;
          WorkingDirectory = cfg.dataDir;
          ExecStart = "${pkgs.s3photoalbum}/bin/thumbnailer";
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

    networking.firewall = mkIf cfg.openFirewall {
      allowedTCPPorts = optional (cfg.nginx != true) 7788
        ++ optional (cfg.nginx) 80 ++ optional (cfg.nginx) 443;
    };

  };
  meta = { maintainers = with lib.maintainers; [ mayniklas ]; };
}
