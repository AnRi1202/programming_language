long f(long n) {
  long sum = 0;
  long i;
  
  // ループ展開のテスト
  for (i = 0; i < n; i = i + 4) {
    if (i < n) sum = sum + i;
    if (i + 1 < n) sum = sum + (i + 1);
    if (i + 2 < n) sum = sum + (i + 2);
    if (i + 3 < n) sum = sum + (i + 3);
  }
  
  return sum;
} 