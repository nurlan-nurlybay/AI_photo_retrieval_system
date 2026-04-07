import 'dart:io';
import 'package:flutter/material.dart';
import 'package:frontend_flutter/data/models.dart';
import 'package:frontend_flutter/core/config.dart';

class CachedImage extends StatelessWidget {
  final String? localPath;
  final String remoteUrl;
  final ImageProcessingStatus? status;
  final BoxFit fit;
  final Widget Function(BuildContext, Widget, ImageChunkEvent?)? loadingBuilder;

  const CachedImage({
    super.key,
    this.localPath,
    required this.remoteUrl,
    this.status,
    this.fit = BoxFit.cover,
    this.loadingBuilder,
  });

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        Positioned.fill(child: _buildImage()),
        if (status != null)
          Positioned(
            top: 6,
            right: 6,
            child: _StatusIndicator(status: status!),
          ),
      ],
    );
  }

  Widget _buildImage() {
    // 1. Check if localPath exists and file is accessible
    if (localPath != null) {
      final file = File(localPath!);
      if (file.existsSync()) {
        return Image.file(
          file,
          fit: fit,
          errorBuilder: (context, error, stackTrace) => _buildNetworkImage(),
        );
      }
    }
    return _buildNetworkImage();
  }

  Widget _buildNetworkImage() {
    return Image.network(
      AppConfig.fixImageUrl(remoteUrl),
      fit: fit,
      loadingBuilder: loadingBuilder,
      errorBuilder: (context, error, stackTrace) => Container(
        color: const Color(0xFF1E1E1E),
        child: const Center(child: Icon(Icons.broken_image, color: Colors.white24)),
      ),
    );
  }
}

class _StatusIndicator extends StatelessWidget {
  final ImageProcessingStatus status;
  const _StatusIndicator({required this.status});

  @override
  Widget build(BuildContext context) {
    Color color;
    switch (status) {
      case ImageProcessingStatus.unprocessed:
        color = Colors.grey;
        break;
      case ImageProcessingStatus.fastEncoded:
        color = Colors.yellow;
        break;
      case ImageProcessingStatus.slowEncoded:
      case ImageProcessingStatus.inIndex:
        color = Colors.green;
        break;
      case ImageProcessingStatus.failed:
        color = Colors.red;
        break;
    }

    return Container(
      width: 10,
      height: 10,
      decoration: BoxDecoration(
        color: color,
        shape: BoxShape.circle,
        border: Border.all(color: Colors.black26, width: 1),
        boxShadow: [
          BoxShadow(color: color.withOpacity(0.4), blurRadius: 4, spreadRadius: 1),
        ],
      ),
    );
  }
}
