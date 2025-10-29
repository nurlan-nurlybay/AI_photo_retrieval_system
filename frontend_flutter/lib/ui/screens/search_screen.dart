import 'dart:io';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import '../../data/api_client.dart';
import '../../data/models.dart';
import '../widgets/image_grid.dart';
import 'package:frontend_flutter/data/api_client.dart';
import 'package:frontend_flutter/data/models.dart';
import 'package:frontend_flutter/ui/widgets/image_grid.dart';
import 'package:frontend_flutter/data/relevance.dart';
import 'package:frontend_flutter/ui/util/pick.dart';

class SearchScreen extends StatefulWidget {
  const SearchScreen({super.key});

  @override
  State<SearchScreen> createState() => _SearchScreenState();
}

class _SearchScreenState extends State<SearchScreen> {
  final _api = ApiClient();
  final _controller = TextEditingController();

  bool _busy = false;
  List<MediaResponse> _results = [];

  File? _probe; // chosen image for similarity search

  Future<void> _doTextSearch() async {
    final q = _controller.text.trim();
    if (q.isEmpty) return;
    setState(() { _busy = true; _results = []; });
    try {
      final res = await _api.textSearch(q);
      final filtered = Relevance.relativeToMax(res.results, alpha: 0.85);
      setState(() => _results = filtered);
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Search failed: $e')));
    } finally {
      setState(() => _busy = false);
    }
  }

Future<void> _pickProbe() async {
  try {
    final f = await pickImageSingle();
    if (!mounted) return;
    if (f == null) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Selection cancelled')));
      return;
    }
    setState(() => _probe = f);
  } catch (e) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Error picking image: $e')));
  }
}

  Future<void> _doImageSearch() async {
    if (_probe == null) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Choose an image first.')));
      return;
    }
    setState(() { _busy = true; _results = []; });
    try {
      final res = await _api.imageSearch(_probe!);
      final filtered = Relevance.relativeToMax(res.results, alpha: 0.6);
      setState(() => _results = filtered);
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Image search failed: $e')));
    } finally {
      setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final border = OutlineInputBorder(borderSide: BorderSide(color: Colors.black.withOpacity(0.6)));
    return Scaffold(
      appBar: AppBar(title: const Text('Text / Image Search')),
      body: Column(
        children: [
          const SizedBox(height: 10),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12.0),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _controller,
                    decoration: InputDecoration(
                      hintText: 'Enter text query...',
                      border: border,
                      enabledBorder: border,
                      focusedBorder: border,
                      isDense: true,
                      contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
                    ),
                    onSubmitted: (_) => _doTextSearch(),
                  ),
                ),
                const SizedBox(width: 8),
                ElevatedButton(
                  onPressed: _busy ? null : _doTextSearch,
                  child: const Text('Search'),
                ),
              ],
            ),
          ),
          const SizedBox(height: 8),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12.0),
            child: Row(
              children: [
                OutlinedButton.icon(
                  onPressed: _busy ? null : _pickProbe,
                  icon: const Icon(Icons.upload_file),
                  label: const Text('Upload Image'),
                ),
                const SizedBox(width: 8),
                ElevatedButton.icon(
                  onPressed: _busy ? null : _doImageSearch,
                  icon: const Icon(Icons.image_search),
                  label: const Text('Image Search'),
                ),
                const Spacer(),
                if (_probe != null)
                  SizedBox(
                    width: 52,
                    height: 52,
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(6),
                      child: Image.file(_probe!, fit: BoxFit.cover),
                    ),
                  ),
              ],
            ),
          ),
          if (_busy) const Padding(
            padding: EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            child: LinearProgressIndicator(),
          ),
          Expanded(child: ImageGrid(items: _results)),
        ],
      ),
    );
  }
}
