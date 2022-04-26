{ lib, pkgs, config, ... }:
with lib;

let cfg = config.services.s3photoalbum-thumbnailer;
in {

  options.services.s3photoalbum-thumbnailer = {

    enable = mkEnableOption "s3photoalbum-thumbnailer";

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

    user = mkOption {
      type = types.str;
      default = "s3photoalbum-thumb";
      description = "User account under which s3photoalbum-thumbnailer runs.";
    };

    group = mkOption {
      type = types.str;
      default = "s3photoalbum-thumb";
      description = "Group under which s3photoalbum-thumbnailer runs.";
    };

  };

  config = mkIf cfg.enable {

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
          Environment = [
            "FFMPEGTHUMBNAILER_PATH='${pkgs.ffmpegthumbnailer}/bin/ffmpegthumbnailer'"
            "THUMBNAIL_SIZE=300"
          ];
        }
        (mkIf (cfg.dataDir == "/var/lib/s3photoalbum") {
          StateDirectory = "s3photoalbum";
        })
      ];
    };

    users.users = mkIf (cfg.user == "s3photoalbum-thumb") {
      s3photoalbum-thumb = {
        isSystemUser = true;
        group = cfg.group;
        description = "s3photoalbum-thumbnailer system user";
      };
    };

    users.groups =
      mkIf (cfg.group == "s3photoalbum-thumb") { s3photoalbum-thumb = { }; };

  };
  meta = { maintainers = with lib.maintainers; [ mayniklas pinpox ]; };
}
