// lib/data/models.dart

/// Processing status of an image in the ML pipeline.
/// Maps to the SSE "status" field from the backend.
enum ImageProcessingStatus {
  /// Just uploaded — not yet searchable.
  unprocessed,
  /// SigLIP fast-encoded — basic similarity search works.
  fastEncoded,
  /// Qwen slow-encoded — deep contextual search works.
  slowEncoded,
  /// Fully indexed in Milvus.
  inIndex,
  /// ML worker failed on this image.
  failed;

  static ImageProcessingStatus fromString(String? s) {
    switch (s) {
      case 'fast_encoded': return ImageProcessingStatus.fastEncoded;
      case 'slow_encoded': return ImageProcessingStatus.slowEncoded;
      case 'in_index':     return ImageProcessingStatus.inIndex;
      case 'failed':       return ImageProcessingStatus.failed;
      case 'pending':
      default:             return ImageProcessingStatus.unprocessed;
    }
  }
}

/// Single search hit when doing text/image search.
class MediaResponse {
  final int id;
  final int userId;
  final String url;
  final String thumbUrl;
  final double? score;
  final String? localPath;
  final ImageProcessingStatus status;

  MediaResponse({
    required this.id,
    required this.userId,
    required this.url,
    required this.thumbUrl,
    this.score,
    this.localPath,
    this.status = ImageProcessingStatus.unprocessed,
  });

  factory MediaResponse.fromJson(Map<String, dynamic> json) {
    return MediaResponse(
      id: json['id'] as int,
      userId: json['user_id'] as int,
      url: json['url'] as String,
      thumbUrl: json['thumb_url'] as String,
      score: (json['score'] as num?)?.toDouble(),
      localPath: json['local_path'] as String?,
      status: ImageProcessingStatus.fromString(json['status'] as String?),
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'user_id': userId,
        'url': url,
        'thumb_url': thumbUrl,
        'score': score,
        'local_path': localPath,
        'status': status.name,
      };
}

/// Response for /api/search/text and /api/search/image
class SearchResponse {
  final List<MediaResponse> results;
  final int total;
  /// Whether the Qwen (slow/deep) model was used for this search.
  /// If false, UI can optionally show a "Fast Search Used" badge.
  final bool usedQwen;

  SearchResponse({
    required this.results,
    required this.total,
    this.usedQwen = false,
  });

  factory SearchResponse.fromJson(Map<String, dynamic> json) {
    final list = (json['results'] as List<dynamic>? ?? [])
        .map((e) => MediaResponse.fromJson(e as Map<String, dynamic>))
        .toList();
    return SearchResponse(
      results: list,
      total: json['total'] as int? ?? list.length,
      usedQwen: json['used_qwen'] as bool? ?? false,
    );
  }
}

/// Full stored media item (used for user gallery, upload results, etc.)
class MediaItem {
  final int id;
  final String url;
  final String thumbUrl;
  final String mimeType;
  final int sizeBytes;
  final int width;
  final int height;
  final String checksum;
  final String? takenAt;
  final String createdAt;
  final String? localPath;
  ImageProcessingStatus status;

  MediaItem({
    required this.id,
    required this.url,
    required this.thumbUrl,
    required this.mimeType,
    required this.sizeBytes,
    required this.width,
    required this.height,
    required this.checksum,
    required this.createdAt,
    this.takenAt,
    this.localPath,
    this.status = ImageProcessingStatus.unprocessed,
  });

  factory MediaItem.fromJson(Map<String, dynamic> json) {
    return MediaItem(
      id: json['id'] as int,
      url: json['url'] as String,
      thumbUrl: json['thumb_url'] as String,
      mimeType: json['mime_type'] as String,
      sizeBytes: json['size_bytes'] as int,
      width: json['width'] as int,
      height: json['height'] as int,
      checksum: json['checksum'] as String,
      takenAt: json['taken_at'] as String?,
      createdAt: json['created_at'] as String,
      localPath: json['local_path'] as String?,
      status: ImageProcessingStatus.fromString(json['status'] as String?),
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'url': url,
        'thumb_url': thumbUrl,
        'mime_type': mimeType,
        'size_bytes': sizeBytes,
        'width': width,
        'height': height,
        'checksum': checksum,
        'taken_at': takenAt,
        'created_at': createdAt,
        'local_path': localPath,
        'status': status.name,
      };
}

/// One entry in UploadResponse.results
class UploadResultModel {
  final String filename;
  final String status; // "saved" | "duplicate" | "failed" | "skipped"
  final String? error;
  final MediaItem? media;

  UploadResultModel({
    required this.filename,
    required this.status,
    this.error,
    this.media,
  });

  factory UploadResultModel.fromJson(Map<String, dynamic> json) {
    return UploadResultModel(
      filename: json['filename'] as String,
      status: json['status'] as String,
      error: json['error'] as String?,
      media: json['media'] != null
          ? MediaItem.fromJson(json['media'] as Map<String, dynamic>)
          : null,
    );
  }
}

/// Summary block inside UploadResponse
class UploadSummary {
  final int total;
  final int saved;
  final int duplicates;
  final int failed;

  UploadSummary({
    required this.total,
    required this.saved,
    required this.duplicates,
    required this.failed,
  });

  factory UploadSummary.fromJson(Map<String, dynamic> json) {
    return UploadSummary(
      total: json['total'] as int? ?? 0,
      saved: json['saved'] as int? ?? 0,
      duplicates: json['duplicates'] as int? ?? 0,
      failed: json['failed'] as int? ?? 0,
    );
  }
}

/// Upload response from /api/upload/image
class UploadResponseModel {
  final List<UploadResultModel> results;
  final UploadSummary summary;

  UploadResponseModel({
    required this.results,
    required this.summary,
  });

  factory UploadResponseModel.fromJson(Map<String, dynamic> json) {
    final list = (json['results'] as List<dynamic>? ?? [])
        .map((e) => UploadResultModel.fromJson(e as Map<String, dynamic>))
        .toList();
    return UploadResponseModel(
      results: list,
      summary: UploadSummary.fromJson(
        json['summary'] as Map<String, dynamic>? ?? const {},
      ),
    );
  }
}

/// Response for /api/user/images/list
class UserImagesListResponse {
  final List<MediaItem> results;
  final int total;

  UserImagesListResponse({
    required this.results,
    required this.total,
  });

  factory UserImagesListResponse.fromJson(Map<String, dynamic> json) {
    final list = (json['results'] as List<dynamic>? ?? [])
        .map((e) => MediaItem.fromJson(e as Map<String, dynamic>))
        .toList();
    return UserImagesListResponse(
      results: list,
      total: json['total'] as int? ?? list.length,
    );
  }
}

/// AI-detected photo category.
class PhotoCategory {
  final String name;
  final dynamic icon; // IconData
  final String coverUrl;
  final int count;
  final String query;
  final List<MediaResponse> results;

  PhotoCategory({
    required this.name,
    required this.icon,
    required this.coverUrl,
    required this.count,
    this.query = '',
    this.results = const [],
  });
}

/// A photo stored inside an album.
class AlbumPhoto {
  final int id;
  final String url;
  final String thumbUrl;
  final String? localPath;

  AlbumPhoto({
    required this.id,
    required this.url,
    required this.thumbUrl,
    this.localPath,
  });

  factory AlbumPhoto.fromJson(Map<String, dynamic> json) => AlbumPhoto(
        id: json['id'] as int,
        url: json['url'] as String,
        thumbUrl: json['thumb_url'] as String,
        localPath: json['local_path'] as String?,
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'url': url,
        'thumb_url': thumbUrl,
        'local_path': localPath,
      };

  factory AlbumPhoto.fromMediaItem(MediaItem item) => AlbumPhoto(
        id: item.id,
        url: item.url,
        thumbUrl: item.thumbUrl,
        localPath: item.localPath,
      );
}

/// User-created photo album (persisted in SharedPreferences).
class PhotoAlbum {
  final String id;
  final String name;
  final List<AlbumPhoto> photos;

  PhotoAlbum({
    required this.id,
    required this.name,
    List<AlbumPhoto>? photos,
  }) : photos = photos ?? [];

  String get coverUrl => photos.isNotEmpty
      ? (photos.first.thumbUrl.isNotEmpty
          ? photos.first.thumbUrl
          : photos.first.url)
      : '';

  int get count => photos.length;

  factory PhotoAlbum.fromJson(Map<String, dynamic> json) => PhotoAlbum(
        id: json['id'] as String,
        name: json['name'] as String,
        photos: (json['photos'] as List<dynamic>?)
                ?.map((e) => AlbumPhoto.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [],
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'photos': photos.map((p) => p.toJson()).toList(),
      };
}
