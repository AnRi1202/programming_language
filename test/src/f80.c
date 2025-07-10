struct Point {
  long x;
  long y;
};

long f(long a, long b) {
  struct Point p;
  p.x = a;
  p.y = b;
  return p.x + p.y;
} 