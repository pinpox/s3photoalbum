{ pkgs ? import <nixpkgs> { } }:
with pkgs;
mkShell {
  buildInputs = [
    ffmpeg
    ffmpegthumbnailer
    exiftool
    go
    gcc
  ];

  shellHook = ''
    source ./env
	export THUMBNAIL_SIZE=300
	export FFMPEGTHUMBNAILER_PATH=${ffmpegthumbnailer}/bin/ffmpegthumbnailer
	export EXIFTOOL_PATH=${exiftool}/bin/exiftool
  '';
}
