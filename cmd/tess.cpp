#include <tesseract/baseapi.h>
#include <leptonica/allheaders.h>

extern "C" {
  class TessClient {
  private:
    tesseract::TessBaseAPI *api;
    Pix *image;
  public:
    TessClient()
    {
      api = new tesseract::TessBaseAPI();
    }
    TessClient(char *imgPath)
    {
      image = pixRead(imgPath);
    }
    void setImage(char* imgPath)
    {
      image = pixRead(imgPath);
    }
    char* Exec()
    {
      api->SetImage(image);
      char *outText = api->GetUTF8Text();
      pixDestroy(&image);
      api->End();
      return outText;
    }
  };

  char* tessMem(const void *buf, size_t bufSize, char* languages) {
    char *out;
    tesseract::TessBaseAPI *api = new tesseract::TessBaseAPI();
    // Initialize tesseract-ocr with English, without specifying tessdata path
    if (api->Init(NULL, languages)) {
      fprintf(stderr, "Could not initialize tesseract.\n");
      exit(1);
    }

    Pix *image = pixReadMem((l_uint8 *)buf, bufSize);
    api->SetImage(image);
    out = api->GetUTF8Text();
    api->End();
    pixDestroy(&image);

    return out;
  }

}/* extern "C" */
