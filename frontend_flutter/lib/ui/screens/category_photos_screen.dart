import 'package:flutter/material.dart';
import 'package:frontend_flutter/data/models.dart';
import 'package:frontend_flutter/ui/widgets/cached_image.dart';

class CategoryPhotosScreen extends StatelessWidget {
  final PhotoCategory category;

  const CategoryPhotosScreen({super.key, required this.category});

  @override
  Widget build(BuildContext context) {
    final results = category.results;

    return Scaffold(
      backgroundColor: const Color(0xFF121212),
      appBar: AppBar(
        title: Text('${category.name} (${category.count})'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => Navigator.pop(context),
        ),
      ),
      body: results.isEmpty
          ? Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(category.icon, size: 48, color: Colors.white24),
                  const SizedBox(height: 16),
                  const Text(
                    'No photos in this category yet.\nUpload photos to see them here.',
                    textAlign: TextAlign.center,
                    style: TextStyle(color: Colors.white54, fontSize: 16),
                  ),
                ],
              ),
            )
          : Padding(
              padding: const EdgeInsets.all(4),
              child: GridView.builder(
                itemCount: results.length,
                gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                  crossAxisCount: 3,
                  mainAxisSpacing: 2,
                  crossAxisSpacing: 2,
                ),
                itemBuilder: (context, index) {
                  final item = results[index];
                  return GestureDetector(
                    onTap: () => _openPhoto(context, item),
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(4),
                      child: CachedImage(
                        localPath: item.localPath,
                        remoteUrl: item.thumbUrl.isNotEmpty
                            ? item.thumbUrl
                            : item.url,
                        fit: BoxFit.cover,
                      ),
                    ),
                  );
                },
              ),
            ),
    );
  }

  void _openPhoto(BuildContext context, MediaResponse item) {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => Scaffold(
          backgroundColor: Colors.black,
          extendBodyBehindAppBar: true,
          appBar: AppBar(
            backgroundColor: Colors.transparent,
            leading: IconButton(
              icon: const Icon(Icons.arrow_back_ios_new, color: Colors.white),
              onPressed: () => Navigator.pop(context),
            ),
          ),
          body: Center(
            child: InteractiveViewer(
              minScale: 1.0,
              maxScale: 4.0,
              child: CachedImage(
                localPath: item.localPath,
                remoteUrl: item.url,
                fit: BoxFit.contain,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
