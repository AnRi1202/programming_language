struct TreeNode {
  long value;
  struct TreeNode* left;
  struct TreeNode* right;
};

long sum_tree(struct TreeNode* node) {
  if (node == 0) {
    return 0;
  }
  return node->value + sum_tree(node->left) + sum_tree(node->right);
}

long f(long x) {
  struct TreeNode root, left, right;
  
  root.value = x;
  left.value = x * 2;
  right.value = x * 3;
  
  root.left = &left;
  root.right = &right;
  left.left = 0;
  left.right = 0;
  right.left = 0;
  right.right = 0;
  
  return sum_tree(&root);
} 