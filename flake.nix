{
  description = "TODO";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      with nixpkgs.legacyPackages.${system}; rec {

        packages = flake-utils.lib.flattenTree rec {

          s3photoalbum = buildGoModule rec {

            pname = "s3photoalbum";
            version = "0.1";

            src = ./.;
            vendorSha256 = "sha256-UG17gs2+AWmpYd1qDEVQ3byu2X128a3SXqC1PKTccH4=";
            subPackages = [ "." ];

            meta = with lib; {
              description = "TODO";
              homepage = "https://github.com/pinpox/s3photoalbum";
              license = licenses.gpl3;
              maintainers = with maintainers; [ pinpox ];
              platforms = platforms.linux;
            };
          };
        };

        defaultPackage = packages.s3photoalbum;
      });
}
