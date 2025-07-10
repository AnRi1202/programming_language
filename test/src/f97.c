long f(long x) {
  long arr[3][2];
  long i, j;
  long sum;
  
  // 配列の初期化
  for (i = 0; i < 3; i = i + 1) {
    for (j = 0; j < 2; j = j + 1) {
      arr[i][j] = x + i * 10 + j;
    }
  }
  
  // 配列の要素を合計
  sum = 0;
  for (i = 0; i < 3; i = i + 1) {
    for (j = 0; j < 2; j = j + 1) {
      sum = sum + arr[i][j];
    }
  }
  
  return sum;
} 