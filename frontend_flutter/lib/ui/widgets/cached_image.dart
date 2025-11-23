import 'dart:io';
import 'package:flutter/material.dart';
import 'package:frontend_flutter/core/config.dart';

class CachedImage extends StatelessWidget {
  final String? localPath;
  final String remoteUrl;
  final BoxFit fit;
  final Widget Function(BuildContext, Widget, ImageChunkEvent?)? loadingBuilder;

  const CachedImage({
    super.key,
    this.localPath,
    required this.remoteUrl,
    this.fit = BoxFit.cover,
    this.loadingBuilder,
  });

  @override
  Widget build(BuildContext context) {
    // 1. Check if localPath exists and file is accessible
    if (localPath != null) {
      final file = File(localPath!);
      if (file.existsSync()) {
        return Image.file(
          file,
          fit: fit,
          errorBuilder: (context, error, stackTrace) {
            // Fallback if file exists but is corrupted
            return _buildNetworkImage();
          },
        );
      }
    }

    // 2. Fallback to network image
    return _buildNetworkImage();
  }

  Widget _buildNetworkImage() {
    return Image.network(
      AppConfig.fixImageUrl(remoteUrl),
      fit: fit,
      loadingBuilder: loadingBuilder,
      errorBuilder: (context, error, stackTrace) {
        return Container(
          color: const Color(0xFF1E1E1E),
          child: const Center(
            child: Icon(Icons.broken_image, color: Colors.white24),
          ),
        );
      },
    );
  }
}
