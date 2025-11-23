import 'dart:io';
import 'package:flutter/material.dart';
import 'package:frontend_flutter/data/models.dart';
import 'package:frontend_flutter/core/config.dart';
import 'package:frontend_flutter/ui/widgets/cached_image.dart';
import 'package:http/http.dart' as http;
import 'package:path_provider/path_provider.dart';

class DetailScreen extends StatelessWidget {
  final String imageUrl;
  final MediaItem? mediaItem;
  final Function(File) onFindSimilar;

  const DetailScreen({
    super.key,
    required this.imageUrl,
    this.mediaItem,
    required this.onFindSimilar,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      extendBodyBehindAppBar: true,
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        leading: IconButton(
          icon: const Icon(Icons.arrow_back_ios_new, color: Colors.white),
          onPressed: () => Navigator.pop(context),
        ),
      ),
      body: Stack(
        fit: StackFit.expand,
        children: [
          // Immersive Image
          InteractiveViewer(
            minScale: 1.0,
            maxScale: 4.0,
            child: Center(
              child: Hero(
                tag: imageUrl,
                child: CachedImage(
                  localPath: mediaItem?.localPath,
                  remoteUrl: imageUrl,
                  fit: BoxFit.contain,
                  loadingBuilder: (context, child, progress) {
                    if (progress == null) return child;
                    return const Center(
                      child: CircularProgressIndicator(color: Colors.white24),
                    );
                  },
                ),
              ),
            ),
          ),

          // Bottom Overlay
          Positioned(
            left: 0,
            right: 0,
            bottom: 0,
            child: Container(
              padding: const EdgeInsets.fromLTRB(24, 24, 24, 48),
              decoration: const BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.bottomCenter,
                  end: Alignment.topCenter,
                  colors: [
                    Colors.black87,
                    Colors.transparent,
                  ],
                ),
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  // Info Button
                  IconButton(
                    onPressed: () => _showInfoSheet(context),
                    icon: const Icon(Icons.info_outline, color: Colors.white70, size: 28),
                    tooltip: 'Info',
                  ),

                  // Find Similar Button
                  ElevatedButton.icon(
                    onPressed: () => _handleFindSimilar(context),
                    icon: const Icon(Icons.image_search, size: 20),
                    label: const Text('Find Similar'),
                    style: ElevatedButton.styleFrom(
                      backgroundColor: Colors.white,
                      foregroundColor: Colors.black,
                      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(30),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  void _showInfoSheet(BuildContext context) {
    showModalBottomSheet(
      context: context,
      backgroundColor: const Color(0xFF1E1E1E),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) => Container(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Image Details',
              style: Theme.of(context).textTheme.titleLarge?.copyWith(color: Colors.white),
            ),
            const SizedBox(height: 16),
            _buildInfoRow('Date', mediaItem?.createdAt ?? 'Unknown'),
            _buildInfoRow('Size', mediaItem != null ? '${(mediaItem!.sizeBytes / 1024 / 1024).toStringAsFixed(2)} MB' : 'Unknown'),
            _buildInfoRow('Dimensions', mediaItem != null ? '${mediaItem!.width} x ${mediaItem!.height}' : 'Unknown'),
            const SizedBox(height: 24),
          ],
        ),
      ),
    );
  }

  Widget _buildInfoRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        children: [
          SizedBox(
            width: 100,
            child: Text(
              label,
              style: const TextStyle(color: Colors.white54),
            ),
          ),
          Text(
            value,
            style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w500),
          ),
        ],
      ),
    );
  }

  Future<void> _handleFindSimilar(BuildContext context) async {
    try {
      // Show loading indicator
      showDialog(
        context: context,
        barrierDismissible: false,
        builder: (ctx) => const Center(child: CircularProgressIndicator()),
      );

      // Use local file if available, otherwise download
      File searchFile;
      if (mediaItem?.localPath != null && File(mediaItem!.localPath!).existsSync()) {
        searchFile = File(mediaItem!.localPath!);
      } else {
        final response = await http.get(Uri.parse(AppConfig.fixImageUrl(imageUrl)));
        final tempDir = await getTemporaryDirectory();
        searchFile = File('${tempDir.path}/temp_search_image.jpg');
        await searchFile.writeAsBytes(response.bodyBytes);
      }

      if (context.mounted) {
        Navigator.pop(context); // Dismiss loading
        Navigator.pop(context); // Close detail screen
        onFindSimilar(searchFile); // Trigger search on Home
      }
    } catch (e) {
      if (context.mounted) {
        Navigator.pop(context); // Dismiss loading
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error: $e')),
        );
      }
    }
  }
}
