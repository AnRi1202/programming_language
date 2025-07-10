long f(long x) {
  long* ptr;
  long y;
  y = x + 10;
  ptr = &y;
  return *ptr;
} 