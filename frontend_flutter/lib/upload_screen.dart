import 'dart:io';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'api_service.dart';

class UploadScreen extends StatefulWidget {
  const UploadScreen({super.key});

  @override
  State<UploadScreen> createState() => _UploadScreenState();
}

class _UploadScreenState extends State<UploadScreen> {
  final picker = ImagePicker();
  List<File> _selectedImages = [];
  String _status = '';

  Future<void> pickImages() async {
    final picked = await picker.pickMultiImage();
    if (picked.isNotEmpty) {
      setState(() => _selectedImages = picked.map((x) => File(x.path)).toList());
    }
  }

  Future<void> uploadImages() async {
    try {
      setState(() => _status = "Uploading...");
      final res = await ApiService.uploadImages(_selectedImages);
      setState(() => _status =
          "Uploaded: ${res['summary']['saved']} saved, ${res['summary']['duplicates']} duplicates");
    } catch (e) {
      setState(() => _status = "Error: $e");
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Upload Images')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            ElevatedButton.icon(
              onPressed: pickImages,
              icon: const Icon(Icons.photo_library),
              label: const Text('Pick Images'),
            ),
            const SizedBox(height: 12),
            ElevatedButton.icon(
              onPressed: uploadImages,
              icon: const Icon(Icons.upload),
              label: const Text('Upload'),
            ),
            const SizedBox(height: 20),
            Text(_status),
            const SizedBox(height: 12),
            Expanded(
              child: GridView.count(
                crossAxisCount: 3,
                children: _selectedImages.map((f) => Image.file(f)).toList(),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
