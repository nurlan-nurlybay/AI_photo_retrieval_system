import 'package:flutter/material.dart';
import 'package:cached_network_image/cached_network_image.dart';
import 'package:image_picker/image_picker.dart';
import 'dart:io';
import '../services/api_service.dart';
import '../models/api_models.dart';

class SearchScreen extends StatefulWidget {
  const SearchScreen({super.key});

  @override
  State<SearchScreen> createState() => _SearchScreenState();
}

class _SearchScreenState extends State<SearchScreen> {
  final TextEditingController _searchController = TextEditingController();
  List<SearchResult> _searchResults = [];
  bool _isSearching = false;
  String _lastQuery = '';
  String _status = '';
  
  // Image search variables
  File? _selectedImage;
  final ImagePicker _picker = ImagePicker();
  double _similarityThreshold = 0.2;

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  Future<void> _performSearch() async {
    final query = _searchController.text.trim();
    if (query.isEmpty) {
      _showErrorSnackBar('Please enter a search query');
      return;
    }

    setState(() {
      _isSearching = true;
      _status = 'Searching for "$query"...';
      _searchResults.clear();
    });

    try {
      final results = await ApiService.searchText(query, similarityThreshold: _similarityThreshold);
      setState(() {
        _searchResults = results;
        _lastQuery = query;
        _status = results.isEmpty
            ? 'No results found for "$query"'
            : 'Found ${results.length} result${results.length == 1 ? '' : 's'} for "$query"';
      });
    } catch (e) {
      setState(() {
        _status = 'Search failed: $e';
        _searchResults.clear();
      });
      _showErrorSnackBar('Search failed: $e');
    } finally {
      setState(() {
        _isSearching = false;
      });
    }
  }

  Future<void> _pickImage() async {
    try {
      final XFile? image = await _picker.pickImage(source: ImageSource.gallery);
      if (image != null) {
        setState(() => _selectedImage = File(image.path));
      }
    } catch (e) {
      _showErrorSnackBar('Error picking image: $e');
    }
  }

  Future<void> _searchByImage() async {
    if (_selectedImage == null) return;
    
    setState(() {
      _isSearching = true;
      _status = 'Searching with image...';
      _searchResults.clear();
    });

    try {
      final results = await ApiService.searchByImage(_selectedImage!, similarityThreshold: _similarityThreshold);
      setState(() {
        _searchResults = results;
        _lastQuery = 'image search';
        _status = results.isEmpty
            ? 'No similar images found'
            : 'Found ${results.length} similar image${results.length == 1 ? '' : 's'}';
      });
    } catch (e) {
      setState(() {
        _status = 'Image search failed: $e';
        _searchResults.clear();
      });
      _showErrorSnackBar('Image search failed: $e');
    } finally {
      setState(() {
        _isSearching = false;
      });
    }
  }

  void _clearSearch() {
    setState(() {
      _searchController.clear();
      _searchResults.clear();
      _status = '';
      _lastQuery = '';
      _selectedImage = null;
    });
  }

  void _showErrorSnackBar(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: Colors.red,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Search Images'),
        backgroundColor: Colors.blue,
        foregroundColor: Colors.white,
        actions: [
          if (_searchResults.isNotEmpty)
            IconButton(
              onPressed: _clearSearch,
              icon: const Icon(Icons.clear),
              tooltip: 'Clear search',
            ),
        ],
      ),
      body: Column(
        children: [
          // Search input section
          Container(
            padding: const EdgeInsets.all(16.0),
            decoration: BoxDecoration(
              color: Colors.grey[50],
              border: Border(
                bottom: BorderSide(color: Colors.grey[300]!),
              ),
            ),
            child: Column(
              children: [
                TextField(
                  controller: _searchController,
                  decoration: InputDecoration(
                    hintText: 'Search for images (e.g., "cat pictures", "beach sunset")',
                    prefixIcon: const Icon(Icons.search),
                    suffixIcon: _searchController.text.isNotEmpty
                        ? IconButton(
                            onPressed: () {
                              _searchController.clear();
                              setState(() {});
                            },
                            icon: const Icon(Icons.clear),
                          )
                        : null,
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(12),
                    ),
                    filled: true,
                    fillColor: Colors.white,
                  ),
                  onSubmitted: (_) => _performSearch(),
                  onChanged: (value) => setState(() {}),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: ElevatedButton.icon(
                    onPressed: _isSearching ? null : _performSearch,
                    icon: _isSearching
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                            ),
                          )
                        : const Icon(Icons.search),
                    label: Text(_isSearching ? 'Searching...' : 'Search by Text'),
                    style: ElevatedButton.styleFrom(
                      backgroundColor: Colors.blue,
                      foregroundColor: Colors.white,
                      padding: const EdgeInsets.symmetric(vertical: 12),
                    ),
                  ),
                ),
                
                const SizedBox(height: 16),
                const Divider(),
                const SizedBox(height: 16),
                
                // Image Search Section
                const Text(
                  'Search by Image',
                  style: TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.bold,
                    color: Colors.grey,
                  ),
                ),
                const SizedBox(height: 12),
                
                // Similarity Threshold
                Row(
                  children: [
                    const Text('Similarity: '),
                    Expanded(
                      child: Slider(
                        value: _similarityThreshold,
                        min: 0.0,
                        max: 1.0,
                        divisions: 20,
                        label: '${(_similarityThreshold * 100).round()}%',
                        onChanged: (value) {
                          setState(() => _similarityThreshold = value);
                        },
                      ),
                    ),
                    Text('${(_similarityThreshold * 100).round()}%'),
                  ],
                ),
                
                const SizedBox(height: 12),
                
                // Image picker and search buttons
                Row(
                  children: [
                    Expanded(
                      child: ElevatedButton.icon(
                        onPressed: _pickImage,
                        icon: const Icon(Icons.image),
                        label: const Text('Pick Image'),
                        style: ElevatedButton.styleFrom(
                          backgroundColor: Colors.green,
                          foregroundColor: Colors.white,
                          padding: const EdgeInsets.symmetric(vertical: 12),
                        ),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: ElevatedButton.icon(
                        onPressed: _selectedImage != null && !_isSearching ? _searchByImage : null,
                        icon: const Icon(Icons.search),
                        label: const Text('Search by Image'),
                        style: ElevatedButton.styleFrom(
                          backgroundColor: Colors.orange,
                          foregroundColor: Colors.white,
                          padding: const EdgeInsets.symmetric(vertical: 12),
                        ),
                      ),
                    ),
                  ],
                ),
                
                // Selected image preview
                if (_selectedImage != null) ...[
                  const SizedBox(height: 16),
                  Container(
                    height: 120,
                    width: 120,
                    decoration: BoxDecoration(
                      border: Border.all(color: Colors.grey),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(8),
                      child: Image.file(_selectedImage!, fit: BoxFit.cover),
                    ),
                  ),
                ],
              ],
            ),
          ),

          // Status section
          if (_status.isNotEmpty)
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(12),
              margin: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: _status.contains('failed') || _status.contains('Error')
                    ? Colors.red[50]
                    : _status.contains('Found') || _status.contains('No results')
                        ? Colors.blue[50]
                        : Colors.grey[100],
                borderRadius: BorderRadius.circular(8),
                border: Border.all(
                  color: _status.contains('failed') || _status.contains('Error')
                      ? Colors.red[200]!
                      : _status.contains('Found') || _status.contains('No results')
                          ? Colors.blue[200]!
                          : Colors.grey[300]!,
                ),
              ),
              child: Text(
                _status,
                style: TextStyle(
                  color: _status.contains('failed') || _status.contains('Error')
                      ? Colors.red[800]
                      : _status.contains('Found') || _status.contains('No results')
                          ? Colors.blue[800]
                          : Colors.black87,
                  fontWeight: FontWeight.w500,
                ),
                textAlign: TextAlign.center,
              ),
            ),

          // Results section
          Expanded(
            child: _searchResults.isEmpty
                ? Center(
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(
                          _lastQuery.isEmpty ? Icons.search : Icons.image_not_supported,
                          size: 64,
                          color: Colors.grey[400],
                        ),
                        const SizedBox(height: 16),
                        Text(
                          _lastQuery.isEmpty
                              ? 'Enter a search query above'
                              : 'No images found',
                          style: TextStyle(
                            fontSize: 18,
                            color: Colors.grey[600],
                          ),
                        ),
                        const SizedBox(height: 8),
                        Text(
                          _lastQuery.isEmpty
                              ? 'Try searching for "cat pictures" or "beach sunset"'
                              : 'Try a different search term',
                          style: TextStyle(
                            color: Colors.grey[500],
                          ),
                          textAlign: TextAlign.center,
                        ),
                      ],
                    ),
                  )
                : GridView.builder(
                    padding: const EdgeInsets.all(16),
                    gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                      crossAxisCount: 2,
                      crossAxisSpacing: 12,
                      mainAxisSpacing: 12,
                      childAspectRatio: 1.0,
                    ),
                    itemCount: _searchResults.length,
                    itemBuilder: (context, index) {
                      final result = _searchResults[index];
                      return _buildImageCard(result);
                    },
                  ),
          ),
        ],
      ),
    );
  }

  Widget _buildImageCard(SearchResult result) {
    return Card(
      elevation: 4,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Expanded(
              child: CachedNetworkImage(
                imageUrl: result.thumbUrl,
                fit: BoxFit.cover,
                placeholder: (context, url) => Container(
                  color: Colors.grey[200],
                  child: const Center(
                    child: CircularProgressIndicator(),
                  ),
                ),
                errorWidget: (context, url, error) => Container(
                  color: Colors.grey[200],
                  child: const Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(
                        Icons.error,
                        color: Colors.red,
                        size: 32,
                      ),
                      SizedBox(height: 4),
                      Text(
                        'Failed to load',
                        style: TextStyle(
                          color: Colors.red,
                          fontSize: 12,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
            Container(
              padding: const EdgeInsets.all(8),
              color: Colors.white,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'ID: ${result.id}',
                    style: const TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    'User: ${result.userId}',
                    style: TextStyle(
                      fontSize: 10,
                      color: Colors.grey[600],
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
